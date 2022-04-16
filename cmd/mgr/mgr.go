package mgr

import (
	"fmt"
	"golang.org/x/net/context"
	cleanCmd "idefav-proxy/clean-iptables/pkg/cmd"
	cleanConfig "idefav-proxy/clean-iptables/pkg/config"
	"idefav-proxy/cmd/server"
	"idefav-proxy/cmd/upgrade"
	"idefav-proxy/iptables/pkg/capture"
	"idefav-proxy/iptables/pkg/config"
	dep "idefav-proxy/iptables/pkg/dependencies"
	"idefav-proxy/iptables/pkg/validation"
	idefavLog "idefav-proxy/pkg/log"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"time"
)

type ManagementServer struct {
	Server http.Server
}

func NewManagementServer(server http.Server) *ManagementServer {
	return &ManagementServer{Server: server}
}

func (m *ManagementServer) Startup() error {
	ln, err := upgrade.Upgrade.Listen("tcp", ":18080")
	if err != nil {
		return err
	}

	go func() {
		err = m.Server.Serve(ln)
		if err != http.ErrServerClosed {
			log.Println("HTTP pserver:", err)
		}
	}()

	return err
}

func (m *ManagementServer) Shutdown() error {
	return m.Server.Shutdown(context.Background())
}

var httpMux *http.ServeMux

func init() {

	httpMux = http.NewServeMux()
	httpMux.HandleFunc("/debug/pprof/", pprof.Index)
	httpMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	httpMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	httpMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	httpMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	httpMux.HandleFunc("/version", func(rw http.ResponseWriter, r *http.Request) {
		//log.Println(version)
		rw.Write([]byte(Version + "\n"))
	})

	httpMux.HandleFunc("/demo", func(rw http.ResponseWriter, r *http.Request) {
		//log.Println(version)
		time.Sleep(100)
		rw.Write([]byte(Version + "\n"))
	})

	httpMux.HandleFunc("/upgrade", func(rw http.ResponseWriter, r *http.Request) {
		log.Println("upgraded")
		err := upgrade.Upgrade.Upgrade()
		if err != nil {
			log.Println("Upgrade failed:", err)
			rw.Write([]byte("upgraded failed:" + err.Error()))
		} else {
			rw.Write([]byte("upgraded success"))
		}

	})

	httpMux.HandleFunc("/download", func(writer http.ResponseWriter, request *http.Request) {
		request.ParseForm()
		ver := request.Form.Get("ver")
		err := Download("idefav-proxy", ver)
		if err != nil {
			writer.Write([]byte(err.Error()))
			return
		}
		writer.Write([]byte("更新成功"))
	})

	httpMux.HandleFunc("/clean-iptables", func(writer http.ResponseWriter, request *http.Request) {
		cfg := &cleanConfig.Config{
			DryRun:             false,
			ProxyUID:           "",
			ProxyGID:           "",
			RedirectDNS:        false,
			DNSServersV4:       nil,
			DNSServersV6:       nil,
			CaptureAllDNS:      false,
			OwnerGroupsInclude: "*",
			OwnerGroupsExclude: "",
		}
		if err := cfg.Validate(); err != nil {
			writer.Write([]byte(err.Error()))
			return
		}
		ext := cleanCmd.NewDependencies(cfg)
		cleaner := cleanCmd.NewIptablesCleaner(cfg, ext)
		cleaner.Run()
		writer.Write([]byte("操作成功"))
	})

	httpMux.HandleFunc("/iptables", func(writer http.ResponseWriter, request *http.Request) {
		cfg := &config.Config{
			DryRun:                  false,
			TraceLogging:            false,
			RestoreFormat:           false,
			ProxyPort:               "15001",
			InboundCapturePort:      "15006",
			InboundTunnelPort:       "15008",
			ProxyUID:                "",
			ProxyGID:                "",
			InboundInterceptionMode: "",
			InboundTProxyMark:       "1337",
			InboundTProxyRouteTable: "133",
			InboundPortsInclude:     "*",
			InboundPortsExclude:     "8080",
			OwnerGroupsInclude:      "*",
			OwnerGroupsExclude:      "",
			OutboundPortsInclude:    "",
			OutboundPortsExclude:    "",
			OutboundIPRangesInclude: "*",
			OutboundIPRangesExclude: "",
			KubeVirtInterfaces:      "",
			ExcludeInterfaces:       "",
			IptablesProbePort:       uint16(15002),
			ProbeTimeout:            5 * time.Second,
			SkipRuleApply:           false,
			RunValidation:           false,
			RedirectDNS:             false,
			DropInvalid:             false,
			CaptureAllDNS:           false,
			OutputPath:              "",
			NetworkNamespace:        "",
			CNIMode:                 false,
		}
		if err := cfg.Validate(); err != nil {
			writer.Write([]byte(err.Error()))
			return
		}
		var ext dep.Dependencies
		if cfg.DryRun {
			ext = &dep.StdoutStubDependencies{}
		} else {
			ext = &dep.RealDependencies{
				CNIMode:          cfg.CNIMode,
				NetworkNamespace: cfg.NetworkNamespace,
			}
		}

		iptConfigurator := capture.NewIptablesConfigurator(cfg, ext)
		if !cfg.SkipRuleApply {
			iptConfigurator.Run()
			if err := capture.ConfigureRoutes(cfg, ext); err != nil {
				idefavLog.Errorf("failed to configure routes: ")
				writer.Write([]byte(err.Error()))
				return
			}
		}
		if cfg.RunValidation {
			hostIP, err := getLocalIP()
			if err != nil {
				// Assume it is not handled by istio-cni and won't reuse the ValidationErrorCode
				panic(err)
			}
			validator := validation.NewValidator(cfg, hostIP)

			if err := validator.Run(); err != nil {
				writer.Write([]byte(err.Error()))
				return
			}
		}
		writer.Write([]byte("操作成功"))
	})

	iServer := http.Server{
		Handler: httpMux,
	}
	var idefavMgrServer = NewManagementServer(iServer)
	server.RegisterServer(idefavMgrServer)
}

func getLocalIP() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() && !ipnet.IP.IsLinkLocalMulticast() {
			return ipnet.IP, nil
		}
	}
	return nil, fmt.Errorf("no valid local IP address found")
}
