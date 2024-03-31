package main

import (
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/dreamscached/minequery/v2"
	_ "github.com/glebarez/go-sqlite"
	"github.com/maelvls/foncia/logutil"
	"github.com/sethgrid/gencurl"
)

var (
	// EnableDebug enables debugFlag logs.
	debugFlag = flag.Bool("debug", false, "Enable debug logs, including equivalent curl commands.")

	serveBasePath = flag.String("basepath", "", "Base path to serve the API on. For example, if set to /api, the API will be served on /api/interventions. Useful for reverse proxies. Must start with a slash.")
	serveAddr     = flag.String("addr", "0.0.0.0:8080", "Address and port to serve the server on.")

	mcHost    = flag.String("mc-host", "", "The hostname of the Minecraft server. Example: myserver.net")
	mcPort    = flag.Uint("mc-port-java", 25565, "The port of the Minecraft Java server.")
	mcBedrock = flag.Uint("mc-port-bedrock", 19132, "The port of the Minecraft Bedrock server.")

	versionFlag = flag.Bool("version", false, "Print the version and exit.")
)

var (
	// version is the version of the binary. It is set at build time.
	version = "unknown"
	date    = "unknown"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				version = setting.Value
			}
			if setting.Key == "vcs.time" {
				date = setting.Value
			}
		}
	}
}

func main() {
	flag.Parse()
	if *debugFlag {
		logutil.EnableDebug = true
		logutil.Debugf("debug output enabled")
	}

	switch flag.Arg(0) {
	case "version":
		fmt.Println(version)
	case "serve":
		if *serveBasePath != "" && !strings.HasPrefix(*serveBasePath, "/") {
			logutil.Errorf("basepath must start with a slash")
			os.Exit(1)
		}
		logutil.Infof("version: %s (%s)", version, date)

		ServeCmd(*serveAddr, *serveBasePath, *mcHost, int(*mcPort), int(*mcBedrock))
	case "list-mc":
		res, err := minequery.Ping17(*mcHost, int(*mcPort))
		if err != nil {
			panic(err)
		}
		fmt.Println(res)
	case "":
		logutil.Errorf("no command given. Use one of: serve, list, token, list-mc, version")
	default:
		logutil.Errorf("unknown command %q", flag.Arg(0))
		os.Exit(1)
	}
}

type tmlpErrData struct {
	Error   string
	Version string
}

var tmlpErr = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
<head>
<title>Error</title>
<meta charset="utf-8">
</head>
<body>
	<h1>Error</h1>
	<p>{{.Error}}</p>
	<div>
		<small>Version: {{.Version}}</small>
	</div>
</body>
</html>
`))

func ServeCmd(serveAddr, basePath, mcHost string, mcPortJava, mcPortBedrock int) {
	if !strings.HasPrefix(basePath, "/") {
		logutil.Errorf("basepath must start with a slash")
		os.Exit(1)
	}
	defaultPath := basePath + "/minecraft"

	client := &http.Client{}
	enableDebugCurlLogs(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(302)
		w.Header().Set("Location", defaultPath)
		tmlpErr.Execute(w, tmlpErrData{
			Error:   fmt.Sprintf(`The actual page is %s`, defaultPath),
			Version: version,
		})
	})

	mux.HandleFunc("/minecraft", ServeMinecraft(mcHost, mcPortJava, mcPortBedrock))

	listner, err := net.Listen("tcp", serveAddr)
	if err != nil {
		logutil.Errorf("while listening: %v", err)
		os.Exit(1)
	}
	logutil.Infof("listening on %s", listner.Addr())
	logutil.Infof("url: http://%v%s", listner.Addr(), defaultPath)

	err = http.Serve(listner, mux)
	if err != nil {
		logutil.Errorf("while listening: %v", err)
		os.Exit(1)
	}
}

func enableDebugCurlLogs(client *http.Client) {
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}
	client.Transport = transportCurlLogs{trWrapped: client.Transport}
}

// Only used when --debug is passed.
type transportCurlLogs struct {
	trWrapped http.RoundTripper
}

func (tr transportCurlLogs) RoundTrip(r *http.Request) (*http.Response, error) {
	logutil.Debugf("%s", gencurl.FromRequest(r))
	return tr.trWrapped.RoundTrip(r)
}
