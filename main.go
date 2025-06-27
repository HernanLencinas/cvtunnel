package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	cvclient "github.com/HernanLencinas/cvtunnel/client"
	cvserver "github.com/HernanLencinas/cvtunnel/server"
	cvshare "github.com/HernanLencinas/cvtunnel/share"
	"github.com/HernanLencinas/cvtunnel/share/ccrypto"
	"github.com/HernanLencinas/cvtunnel/share/cos"
	"github.com/HernanLencinas/cvtunnel/share/settings"
)

var banner = `

 ██████╗██╗      ██████╗ ██╗   ██╗██████╗ ██╗   ██╗ █████╗ ██╗     ██╗     ███████╗██╗   ██╗
██╔════╝██║     ██╔═══██╗██║   ██║██╔══██╗██║   ██║██╔══██╗██║     ██║     ██╔════╝╚██╗ ██╔╝
██║     ██║     ██║   ██║██║   ██║██║  ██║██║   ██║███████║██║     ██║     █████╗   ╚████╔╝ 
██║     ██║     ██║   ██║██║   ██║██║  ██║╚██╗ ██╔╝██╔══██║██║     ██║     ██╔══╝    ╚██╔╝  
╚██████╗███████╗╚██████╔╝╚██████╔╝██████╔╝ ╚████╔╝ ██║  ██║███████╗███████╗███████╗   ██║   
 ╚═════╝╚══════╝ ╚═════╝  ╚═════╝ ╚═════╝   ╚═══╝  ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝   ╚═╝   
                                                                                            
████████╗██╗   ██╗███╗   ██╗███╗   ██╗███████╗██╗                                           
╚══██╔══╝██║   ██║████╗  ██║████╗  ██║██╔════╝██║                                           
   ██║   ██║   ██║██╔██╗ ██║██╔██╗ ██║█████╗  ██║                                           
   ██║   ██║   ██║██║╚██╗██║██║╚██╗██║██╔══╝  ██║                                           
   ██║   ╚██████╔╝██║ ╚████║██║ ╚████║███████╗███████╗                                      
   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═══╝╚══════╝╚══════╝          

`

var help = banner + `                                 
  Uso: cvtun [commando]

  Comandos:
  
    server - Modo servidor
    client - Modo cliente

`

func main() {

	version := flag.Bool("version", false, "")
	flag.Usage = func() {}
	flag.Parse()

	if *version {
		fmt.Println(cvshare.BuildVersion)
		os.Exit(0)
	}

	args := flag.Args()

	subcmd := ""
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	switch subcmd {
	case "server":
		server(args)
	case "client":
		client(args)
	default:
		fmt.Print(help)
		os.Exit(0)
	}
}

var commonHelp = `
    --help, This help text

`

func generatePidFile() {
	pid := []byte(strconv.Itoa(os.Getpid()))
	if err := os.WriteFile("cvtunnel.pid", pid, 0644); err != nil {
		log.Fatal(err)
	}
}

var serverHelp = banner + `
  Uso: cvtun server [opciones]

  Opciones:

    --host

      Define el host de escucha HTTP, es decir, la interfaz de red.
	  (Por defecto usa la variable de entorno HOST y si no está definida, usa 0.0.0.0).

    --port, -p

      Define el puerto de escucha HTTP.
	  (Por defecto usa la variable de entorno PORT, y si no está definida, usa el puerto 8000).
	
    --key (obsoleta, usar --keygen y --keyfile en su lugar)

   	  Una cadena opcional para sembrar la generación de un par de claves ECDSA (pública y privada).
	  Todas las comunicaciones se asegurarán usando este par de claves.
	  Compartí la huella digital (fingerprint) resultante con los clientes para detectar posibles ataques de tipo “man-in-the-middle”.
	  (Por defecto usa la variable de entorno CVTUN_KEY, si no está definida se genera una nueva clave en cada ejecución).
	
    --keygen

      Ruta donde guardar una nueva clave privada SSH codificada en formato PEM.
      Si los usuarios dependen de tu huella digital generada con --key, también podés incluir --key para reutilizar tu clave existente.
      Usá - (guión) para imprimir la clave generada por salida estándar (stdout).
	
    --keyfile

      Ruta opcional hacia un archivo de clave privada SSH codificada en PEM.
      Si se especifica esta opción, se ignora --key y se usa la clave proporcionada para asegurar las comunicaciones.
      (Por defecto usa la variable de entorno CVTUN_KEY_FILE).
      Dado que las claves ECDSA son cortas, también se puede usar una clave codificada en base64 directamente en línea (por ejemplo: cvtun server --keygen - | base64).

    --authfile

      Ruta opcional hacia un archivo users.json. Este archivo debe ser un objeto con usuarios definidos así:

  	  {
	    "<usuario:contraseña>": ["<expresión-regular-dirección>", "<expresión-regular-dirección>"]
	  }

      Cuando un <usuario> se conecta, se verifica su <contraseña>, y luego se comparan las direcciones remotas contra la lista de expresiones regulares para ver si coinciden.
      Las direcciones tendrán forma de "host-remoto:puerto" para túneles normales y "R:interfaz-local:puerto" para reenvío de puertos reverso.
      Este archivo se recargará automáticamente si se modifica.

    --auth

      Cadena opcional que representa un único usuario con acceso total, en formato <usuario:contraseña>.
      Es equivalente a un archivo de autenticación con {"<usuario:contraseña>": [""]}.
      Si no se establece, se usará la variable de entorno AUTH.
	
    --keepalive

      Intervalo opcional de keepalive. Dado que el transporte es HTTP, a menudo pasa por proxies que cierran conexiones inactivas.
      Especificá un tiempo con unidad, por ejemplo: 5s o 2m.
      (Por defecto es 25s, y se puede desactivar con 0s).
	
    --backend

      Especifica otro servidor HTTP al cual redirigir las solicitudes HTTP normales.
      Útil para “camuflar” el servidor cvtun.
	
    --socks5

      Permite que los clientes accedan al proxy interno SOCKS5.
      (Ver cvtun client --help para más información).
	
    --reverse

      Permite que los clientes especifiquen túneles de reenvío de puertos reverso además de los normales.
	
    --tls-key

      Habilita TLS y permite especificar la ruta a una clave privada codificada en PEM.
      Cuando se usa esta opción, también se debe usar --tls-cert y no se puede usar --tls-domain.
	
    --tls-cert

      Habilita TLS y permite especificar la ruta a un certificado TLS codificado en PEM.
      Cuando se usa esta opción, también se debe usar --tls-key y no se puede usar --tls-domain.
	
    --tls-domain

      Habilita TLS y obtiene automáticamente un certificado TLS con Let’s Encrypt.
      Requiere que el puerto 443 esté disponible.
      Se pueden especificar múltiples --tls-domain para servir varios dominios.
      Los certificados obtenidos se guardan en "$HOME/.cache/cvtun" (se puede cambiar con la variable CVTUN_LE_CACHE, o desactivar con CVTUN_LE_CACHE=-).
      También se puede establecer un email de notificación con CVTUN_LE_EMAIL.
	
    --tls-ca

      Ruta a un archivo PEM que contenga certificados CA o a un directorio con múltiples archivos PEM.
      Se usa para validar conexiones de clientes.
      Estos certificados reemplazan a los certificados raíz del sistema.
      Es comúnmente usado para implementar TLS mutuo (mTLS).

`

func server(args []string) {

	fmt.Print(banner)
	flags := flag.NewFlagSet("server", flag.ContinueOnError)

	config := &cvserver.Config{}
	flags.StringVar(&config.KeySeed, "key", "", "")
	flags.StringVar(&config.KeyFile, "keyfile", "", "")
	flags.StringVar(&config.AuthFile, "authfile", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	flags.StringVar(&config.Proxy, "backend", "", "")
	flags.BoolVar(&config.Socks5, "socks5", false, "")
	flags.BoolVar(&config.Reverse, "reverse", false, "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.Var(multiFlag{&config.TLS.Domains}, "tls-domain", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")

	host := flags.String("host", "", "")
	p := flags.String("p", "", "")
	port := flags.String("port", "", "")
	pid := flags.Bool("pid", false, "")
	verbose := flags.Bool("v", false, "")
	keyGen := flags.String("keygen", "", "")

	flags.Usage = func() {
		fmt.Print(serverHelp)
		os.Exit(0)
	}
	flags.Parse(args)

	if *keyGen != "" {
		if err := ccrypto.GenerateKeyFile(*keyGen, config.KeySeed); err != nil {
			log.Fatal(err)
		}
		return
	}

	if config.KeySeed != "" {
		log.Print("Option `--key` is deprecated and will be removed in a future version of cvtunnel.")
		log.Print("Please use `cvtunnel server --keygen /file/path`, followed by `cvtunnel server --keyfile /file/path` to specify the SSH private key")
	}

	if *host == "" {
		*host = os.Getenv("HOST")
	}
	if *host == "" {
		*host = "0.0.0.0"
	}
	if *port == "" {
		*port = *p
	}
	if *port == "" {
		*port = os.Getenv("PORT")
	}
	if *port == "" {
		*port = "8000"
	}
	if config.KeyFile == "" {
		config.KeyFile = settings.Env("KEY_FILE")
	} else if config.KeySeed == "" {
		config.KeySeed = settings.Env("KEY")
	}
	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}
	s, err := cvserver.NewServer(config)
	if err != nil {
		log.Fatal(err)
	}
	s.Debug = *verbose
	if *pid {
		generatePidFile()
	}
	go cos.GoStats()
	ctx := cos.InterruptContext()
	if err := s.StartContext(ctx, *host, *port); err != nil {
		log.Fatal(err)
	}
	if err := s.Wait(); err != nil {
		log.Fatal(err)
	}
}

type multiFlag struct {
	values *[]string
}

func (flag multiFlag) Set(arg string) error {
	*flag.values = append(*flag.values, arg)
	return nil
}

func (flag multiFlag) String() string {
	return strings.Join(*flag.values, ", ")
}

type headerFlags struct {
	http.Header
}

func (flag *headerFlags) String() string {
	out := ""
	for k, v := range flag.Header {
		out += fmt.Sprintf("%s: %s\n", k, v)
	}
	return out
}

func (flag *headerFlags) Set(arg string) error {
	index := strings.Index(arg, ":")
	if index < 0 {
		return fmt.Errorf(`Invalid header (%s). Should be in the format "HeaderName: HeaderContent"`, arg)
	}
	if flag.Header == nil {
		flag.Header = http.Header{}
	}
	key := arg[0:index]
	value := arg[index+1:]
	flag.Header.Set(key, strings.TrimSpace(value))
	return nil
}

var clientHelp = banner + `
  Uso: cvtun client [opciones] <servidor> <remoto> [remoto] [remoto] ...

  <servidor>: URL del servidor cvtun.
  <remoto>: Conexiones remotas que serán tunelizadas a través del servidor.

  Cada una debe tener el siguiente formato:

    <host-local>:<puerto-local>:<host-remoto>:<puerto-remoto>/<protocolo>

    • Detalles de cada campo:
    • host-local por defecto es 0.0.0.0 (todas las interfaces).
    • puerto-local por defecto es igual al puerto-remoto.
    • puerto-remoto es obligatorio.
    • host-remoto por defecto es 0.0.0.0 (localhost en el servidor).
    • protocolo por defecto es tcp.

  Esto comparte el <host-remoto>:<puerto-remoto> desde el servidor hacia el cliente como <host-local>:<puerto-local>, o bien:

     R:<interfaz-local>:<puerto-local>:<host-remoto>:<puerto-remoto>/<protocolo>

  Esto realiza reenvío de puerto reverso, compartiendo <host-remoto>:<puerto-remoto> desde el cliente hacia el <interfaz-local>:<puerto-local> del servidor.

  Ejemplos de remotos:

    3000
    example.com:3000
    3000:google.com:80
    192.168.0.5:3000:google.com:80
    socks
    5000:socks
    R:2222:localhost:22
    R:socks
    R:5000:socks
    stdio:example.com:22
    1.1.1.1:53/udp

  Casos especiales:

  Si el servidor cvtun tiene --socks5 habilitado, se puede usar "socks" como host-remoto y puerto-remoto.
  Por defecto se conectará a 127.0.0.1:1080. Las conexiones serán manejadas por el proxy SOCKS5 interno del servidor.

  Si el servidor tiene habilitado --reverse, los túneles que empiecen con R: serán reversos: el servidor escucha y el cliente responde.
  El caso R:socks permite que el servidor escuche en el puerto SOCKS5 (1080) y reenvíe la conexión al proxy SOCKS5 interno del cliente.
  Si se usa stdio como host-local, se conectará el stdin/stdout del programa con el destino remoto.
  Útil con ssh como ProxyCommand, por ejemplo:

  ssh -o ProxyCommand='cvtun client cvtunserver stdio:%h:%p' user@example.com

  Opciones:
    
    --fingerprint

      Altamente recomendado. Cadena de huella digital para validar la clave pública del servidor.
      Si no coincide, la conexión se cerrará.
      La huella se genera con SHA256 sobre la clave pública ECDSA y se codifica en base64.
      Debe tener 44 caracteres y terminar en =.
    
    --auth

      Usuario y contraseña opcionales para autenticación del cliente, en formato <usuario>:<contraseña>.
      Se validan contra los definidos en el archivo --authfile del servidor.
      (Por defecto usa la variable de entorno AUTH).
    
    --keepalive

      Intervalo opcional de keepalive. Recomendado cuando se atraviesan proxies HTTP que pueden cerrar conexiones inactivas.
      Se debe especificar con unidad (5s, 2m, etc.).
      (Por defecto es 25s, se puede desactivar con 0s).
    
    --max-retry-count

      Número máximo de reintentos antes de salir.
      (Por defecto es ilimitado).
    
    --max-retry-interval

      Tiempo máximo de espera entre reintentos después de una desconexión.
      (Por defecto es 5 minutos).

    --proxy

      Proxy opcional HTTP CONNECT o SOCKS5 para conectarse al servidor cvtun.
      Se puede incluir autenticación en la URL, por ejemplo:

        http://admin:password@mi-servidor.com:8081
        socks://admin:password@mi-servidor.com:1080

    --header

      Define un encabezado personalizado en formato "Nombre: Contenido".
      Puede usarse múltiples veces.
      (Ejemplo: --header "Foo: Bar" --header "Hello: World")

    --hostname

      Permite sobrescribir el encabezado Host.
      (Por defecto usa el host de la URL del servidor).

    --sni

      Permite sobrescribir el nombre de servidor (ServerName) al usar TLS.
      (Por defecto usa el hostname).
    
    --tls-ca

      Ruta a un archivo PEM con certificados raíz para validar el servidor cvtun.
      Solo válido si se usa "https" o "wss".
      (Por defecto se usan los certificados del sistema operativo).

    --tls-skip-verify

      Omite la verificación del certificado TLS del servidor (cadena y nombre del host).
      Si se activa, el cliente aceptará cualquier certificado TLS presentado por el servidor.
      Solo aplica a conexiones https o wss.
      Nota: La clave pública del servidor aún puede verificarse con --fingerprint.

    --tls-key

      Ruta a un archivo PEM con clave privada usada para autenticación mutua (mTLS) del cliente.

    --tls-cert

      Ruta a un archivo PEM con el certificado correspondiente a la clave anterior.
      Debe tener habilitada la autenticación de cliente (mTLS).

`

func client(args []string) {
	fmt.Print(banner)
	flags := flag.NewFlagSet("client", flag.ContinueOnError)
	config := cvclient.Config{Headers: http.Header{}}
	flags.StringVar(&config.Fingerprint, "fingerprint", "", "")
	flags.StringVar(&config.Auth, "auth", "", "")
	flags.DurationVar(&config.KeepAlive, "keepalive", 25*time.Second, "")
	flags.IntVar(&config.MaxRetryCount, "max-retry-count", -1, "")
	flags.DurationVar(&config.MaxRetryInterval, "max-retry-interval", 0, "")
	flags.StringVar(&config.Proxy, "proxy", "", "")
	flags.StringVar(&config.TLS.CA, "tls-ca", "", "")
	flags.BoolVar(&config.TLS.SkipVerify, "tls-skip-verify", false, "")
	flags.StringVar(&config.TLS.Cert, "tls-cert", "", "")
	flags.StringVar(&config.TLS.Key, "tls-key", "", "")
	flags.Var(&headerFlags{config.Headers}, "header", "")
	hostname := flags.String("hostname", "", "")
	sni := flags.String("sni", "", "")
	pid := flags.Bool("pid", false, "")
	verbose := flags.Bool("v", false, "")
	flags.Usage = func() {
		fmt.Print(clientHelp)
		os.Exit(0)
	}
	flags.Parse(args)
	args = flags.Args()
	if len(args) < 2 {
		log.Fatalf("A server and least one remote is required")
	}

	config.Server = args[0]
	config.Remotes = args[1:]

	connString := strings.Split(config.Remotes[0], ":")
	fmt.Println("Información de Conexión:")
	if len(connString) == 3 {
		fmt.Printf(" ↳ Local IP/Puerto: %s\n", "localhost:"+connString[0])
		fmt.Printf(" ↳ Remoto IP/Puerto: %s\n\n", connString[1]+":"+connString[2])
	} else if len(connString) >= 4 {
		fmt.Printf(" ↳ Local IP/Puerto: %s\n", connString[0]+":"+connString[1])
		fmt.Printf(" ↳ Remoto IP/Puerto: %s\n\n", connString[2]+":"+connString[3])
	} else {
		fmt.Println("Formato de remoto inválido. Se esperaba al menos 3 o 4 partes separadas por ':'")
	}

	//default auth
	if config.Auth == "" {
		config.Auth = os.Getenv("AUTH")
	}
	//move hostname onto headers
	if *hostname != "" {
		config.Headers.Set("Host", *hostname)
		config.TLS.ServerName = *hostname
	}

	if *sni != "" {
		config.TLS.ServerName = *sni
	}

	//ready
	c, err := cvclient.NewClient(&config)
	if err != nil {
		log.Fatal(err)
	}
	c.Debug = *verbose
	if *pid {
		generatePidFile()
	}
	go cos.GoStats()
	ctx := cos.InterruptContext()
	if err := c.Start(ctx); err != nil {
		log.Fatal(err)
	}
	if err := c.Wait(); err != nil {
		log.Fatal(err)
	}

}
