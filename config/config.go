package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/spf13/viper"
	logging "github.com/vkuznet/http-logging"
)

// OAuthRecord defines OAuth provider's credentials
type OAuthRecord struct {
	Provider     string `mapstructure:"Provider"`     // name of the provider
	ClientID     string `mapstructure:"ClientId"`     // client id
	ClientSecret string `mapstructure:"ClientSecret"` // client secret
}

// Kerberos defines kerberos optinos
type Kerberos struct {
	Krb5Conf string `mapstructure:Krb5Conf`
	Keytab   string `mapstructure:Keytab`
	Realm    string `mapstructure:Realm`
}

// GinOptions controls go-gin specific options
type GinOptions struct {
	ColorConsole bool `mapstructure:"ColorConsole"` // control  gin.DisableConsoleColor()
}

// WebServer represents common web server configuration
type WebServer struct {
	// git server options
	GinOptions `mapstructure:"GinOptions"`

	// basic options
	Port        int    `mapstructure:"Port"`        // server port number
	Verbose     int    `mapstructure:"Verbose"`     // verbose output
	Base        string `mapstructure:"Base"`        // base URL
	StaticDir   string `mapstructure:"StaticDir"`   // speficy static dir location
	LogFile     string `mapstructure:"LogFile"`     // server log file
	LogLongFile bool   `mapstructure:"LogLongFile"` // server log structure

	// middleware server parts
	LimiterPeriod string `mapstructure:"Rate"` // limiter rate value

	// proxy parts
	XForwardedHost      string `mapstructure:"X-Forwarded-Host"`       // X-Forwarded-Host field of HTTP request
	XContentTypeOptions string `mapstructure:"X-Content-Type-Options"` // X-Content-Type-Options option

	// TLS server parts
	RootCAs     string   `mapstructure:"RootCAs"`     // server Root CAs path
	ServerCrt   string   `mapstructure:"ServerCert"`  // server certificate
	ServerKey   string   `mapstructure:"ServerKey"`   // server certificate
	DomainNames []string `mapstructure:"DomainNames"` // LetsEncrypt domain names
}

// Frontend stores frontend configuration parameters
type Frontend struct {
	WebServer `mapstructure:"WebServer"`

	// OAuth parts
	OAuth []OAuthRecord `mapstructure:"OAuth"` // oauth configurations

	// captcha parts
	CaptchaSecretKey string `mapstructure:"CaptchaSecretKey"` // re-captcha secret key
	CaptchaPublicKey string `mapstructure:"CaptchaPublicKey"` // re-captcha public key
	CaptchaVerifyUrl string `mapstructure:"CaptchaVerifyUrl"` // re-captcha verify url

	// cookies parts
	UserCookieExpires int64 `mapstructure:"UserCookieExpires"` // expiration of user cookie

	// other options
	TestMode bool `mapstructure:TestMode` // test mode
}

// Encryption represents encryption configuration parameters
type Encryption struct {
	Secret string `mapstructure:"Secret"`
	Cipher string `mapstructure:"Cipher"`
}

// MongoDB represents MongoDB parameters
type MongoDB struct {
	DBName string `mapstructure:"DBName"` // database name
	DBColl string `mapstructure:"DBColl"` // database collection
	DBUri  string `mapstructure:"DBUri"`  // database URI
}

// Discovery represents discovery service configuration
type Discovery struct {
	WebServer  `mapstructure:"WebServer"`
	MongoDB    `mapstructure:"MongoDB"`
	Encryption `mapstructure:"Encryption"`
}

// MetaData represents metadata service configuration
type MetaData struct {
	WebServer `mapstructure:"WebServer"`
	MongoDB   `mapstructure:"MongoDB"`
}

// CHESSMetaData represents CHESS MetaData configuration
type CHESSMetaData struct {
	WebServer           `mapstructure:"WebServer"`
	MongoDB             `mapstructure:"MongoDB"`
	TestMode            bool                `mapstructure:TestMode`      // test mode
	SchemaFiles         []string            `json:"SchemaFiles"`         // schema files
	SchemaRenewInterval int                 `json:"SchemaRenewInterval"` // schema renew interval
	SchemaSections      []string            `json:"SchemaSections"`      // logical schema section list
	WebSectionKeys      map[string][]string `json:"WebSectionKeys"`      // section order dict
}

// OreCastMetaData represents OreCast MetaData configuration
type OreCastMetaData struct {
	WebServer `mapstructure:"WebServer"`
	MongoDB   `mapstructure:"MongoDB"`
}

// DataManagement represents data-management service configuration
type DataManagement struct {
	WebServer `mapstructure:"WebServer"`
}

// DataBookkeeping represents data-bookkeeping service configuration
type DataBookkeeping struct {
	WebServer `mapstructure:"WebServer"`

	DBFile             string `mapstructure:"DBFile"`             // dbs db file with secrets
	MaxDBConnections   int    `mapstructure:"MaxDbConnections"`   // maximum number of DB connections
	MaxIdleConnections int    `mapstructure:"MaxIdleConnections"` // maximum number of idle connections
}

// Authz represents authz service configuration
type Authz struct {
	WebServer  `mapstructure:"WebServer"`
	Encryption `mapstructure:"Encryption"`

	TestMode     bool   `mapstructure:TestMode` // test mode
	DBUri        string `mapstructure:"DBUri"`  // database URI
	ClientID     string `mapstructure:"ClientId"`
	ClientSecret string `mapstructure:"ClientSecret"`
	Domain       string `mapstructure:"Domain"`
	TokenExpires int64  `mapstructure:TokenExpires` // expiration of token
}

// Services represents services structure
type Services struct {
	FrontendURL        string `mapstructure:"FrontendUrl"`
	DiscoveryURL       string `mapstructure:"DiscoveryUrl"`
	MetaDataURL        string `mapstructure:"MetaDataUrl"`
	DataManagementURL  string `mapstructure:"DataManagementUrl"`
	DataBookkeepingURL string `mapstructure:"DataBookkeepingUrl"`
	AuthzURL           string `mapstructure:"AuthzUrl"`
}

// SrvConfig represents configuration structure
type SrvConfig struct {
	Frontend        `mapstructure:"Frontend"`
	Discovery       `mapstructure:"Discovery"`
	MetaData        `mapstructure:"MetaData"`
	DataManagement  `mapstructure:"DataManagement"`
	DataBookkeeping `mapstructure:"DataBookkeeping"`
	Authz           `mapstructure:"Authz"`
	Kerberos        `mapstructure:"Kerberos"`
	Services        `mapstructure:"Services"`
	Encryption      `mapstructure:"Encryption"`
	CHESSMetaData   `mapstructure:"CHESSMetaData"`
	OreCastMetaData `mapstructure:"OreCastMetaData"`
}

func ParseConfig(cfile string) (SrvConfig, error) {
	var config SrvConfig
	if cfile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("ERROR", err)
			os.Exit(1)
		}
		// Search config in home directory with name ".srv" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".srv")
		// setup cfile to $HOME/.srv.yaml
		cfile = filepath.Join(home, ".srv.yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		msg := err.Error()
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			msg = fmt.Sprintf("fail to read %s file, error %v", cfile, err)
		} else {
			// Config file was found but another error was produced
			msg = fmt.Sprintf("unable to parse %s, error %v", cfile, err)
		}
		return config, errors.New(msg)
	}
	if err := viper.Unmarshal(&config); err != nil {
		return config, err
	}
	return config, nil
}

/*
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cfile := "srv.json"
	config, err := ParseConfig(cfile)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	fmt.Printf("Frontend %+v\n", config.Frontend)
	fmt.Printf("Authz %+v\n", config.Authz)
}
*/

// Config represnets configuration instance
var Config *SrvConfig

func Info() string {
	goVersion := runtime.Version()
	tstamp := time.Now()
	return fmt.Sprintf("git={{VERSION}} go=%s date=%s", goVersion, tstamp)
}

func Init() {
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	var config string
	flag.StringVar(&config, "config", "", "server config file")
	flag.Parse()
	if version {
		fmt.Println("server version:", Info())
		return
	}
	oConfig, err := ParseConfig(config)
	if err != nil {
		log.Fatal("ERROR", err)
	}
	Config = &oConfig

	// log settings for web servers
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if Config.Frontend.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.Discovery.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.MetaData.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.DataManagement.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.DataBookkeeping.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.Authz.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.CHESSMetaData.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else if Config.OreCastMetaData.WebServer.LogLongFile {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	}

	// setup log rotate if necessary for all web servers
	rotateLogs(Config.Frontend.WebServer.LogFile)
	rotateLogs(Config.Discovery.WebServer.LogFile)
	rotateLogs(Config.MetaData.WebServer.LogFile)
	rotateLogs(Config.DataManagement.WebServer.LogFile)
	rotateLogs(Config.DataBookkeeping.WebServer.LogFile)
	rotateLogs(Config.Authz.WebServer.LogFile)
	rotateLogs(Config.CHESSMetaData.WebServer.LogFile)
	rotateLogs(Config.OreCastMetaData.WebServer.LogFile)
}

// helper function to rotate logs
func rotateLogs(srvLogName string) {
	if srvLogName != "" {
		log.SetOutput(new(logging.LogWriter))
		rl, err := rotatelogs.New(logName(srvLogName))
		if err == nil {
			rotlogs := logging.RotateLogWriter{RotateLogs: rl}
			log.SetOutput(rotlogs)
		}
	}
}

// logName returns proper log name based on Config LogFile and either
// hostname or pod name (used in k8s environment).
func logName(srvLogName string) string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println("unable to get hostname", err)
	}
	if os.Getenv("MY_POD_NAME") != "" {
		hostname = os.Getenv("MY_POD_NAME")
	}
	logName := srvLogName + "_%Y%m%d"
	if hostname != "" {
		logName = fmt.Sprintf("%s_%s", srvLogName, hostname) + "_%Y%m%d"
	}
	return logName
}
