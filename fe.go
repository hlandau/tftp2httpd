package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/hlandau/slogkit/slogtree"
	"github.com/hlandau/slogkit/slogtreecfg"
	"github.com/hlandau/tftpsrv"
	"gopkg.in/hlandau/service.v3"
)

var log, Log = slogtree.NewFacility("tftp2httpd")

var (
	knReqGet                 = log.MakeKnownInfo("REQ_GET")
	knReqGetErrorBadFilename = log.MakeKnownError("REQ_GET_ERROR_BAD_FILENAME")
	knReqGetErrorHTTP        = log.MakeKnownError("REQ_GET_ERROR_HTTP")
)

var re_valid_fn = regexp.MustCompile("^([a-zA-Z0-9_-][a-zA-Z0-9_. :-]*/)*[a-zA-Z0-9_-][a-zA-Z0-9_. :-]*$")

func validateFilename(fn string) bool {
	return re_valid_fn.MatchString(fn)
}

func handler(ctx context.Context, req *tftpsrv.Request) error {
	log.LogCtx(ctx, knReqGet, "filename", req.Filename)
	defer req.Close()

	addr := req.ClientAddress()
	if !validateFilename(req.Filename) {
		req.WriteError(tftpsrv.ErrFileNotFound, "File not found (invalid filename)")
		log.LogCtx(ctx, knReqGetErrorBadFilename, "remoteAddr", addr.IP.String())
		return nil
	}

	hReq, err := http.NewRequest("GET", settings.HttpUrl+req.Filename, nil)
	if err != nil {
		return err
	}

	hReq.Header.Add("X-Forwarded-For", addr.IP.String())
	hReq.Header.Add("User-Agent", "tftp2httpd")
	res, err := http.DefaultClient.Do(hReq)
	if err != nil {
		log.LogCtx(ctx, knReqGetErrorHTTP, "remoteAddr", addr.IP.String(), "filename", req.Filename, "httpError", err)
		return err
	}
	defer res.Body.Close()

	// Don't return error pages.
	if res.StatusCode != 200 {
		req.WriteError(tftpsrv.ErrFileNotFound, "File not found")
		log.LogCtx(ctx, knReqGetErrorHTTP, "remoteAddr", addr.IP.String(), "filename", req.Filename, "httpCode", res.StatusCode)
		return nil
	}

	buf := make([]byte, 512)
	for {
		n, err := res.Body.Read(buf)
		if n > 0 {
			req.Write(buf[0:n])
		}
		if err != nil {
			break
		}
	}

	return nil
}

var settings struct {
	HttpUrl    string             `default:"" help:"HTTP URL prefix to map to"`
	TftpListen string             `default:":69" help:"TFTP address to bind to"`
	Log        slogtreecfg.Config `embed:"" prefix:"log."`
	Service    service.Config     `embed:"" prefix:"service."`
	Version    bool               `help:"Print version information."`
}

func main() {
	kong.Parse(&settings)
	if settings.Version {
		if buildInfo, ok := debug.ReadBuildInfo(); ok {
			fmt.Print(buildInfo.String())
		}
		return
	}

	ctx := context.Background()
	ctx = slogtreecfg.InitConfig(ctx, settings.Log)

	service.Main(&service.Info{
		Name:          "tftp2httpd",
		Description:   "TFTP to HTTP Daemon",
		DefaultChroot: service.EmptyChrootPath,

		Config: settings.Service,

		RunFunc: func(smgr service.Manager) error {
			s := tftpsrv.Server{
				Addr: settings.TftpListen,
				ReadHandler: func(req *tftpsrv.Request) error {
					return handler(ctx, req)
				},
			}

			err := s.Listen()
			if err != nil {
				return err
			}

			err = smgr.DropPrivileges()
			if err != nil {
				return err
			}

			smgr.SetStarted()
			smgr.SetStatus("tftp2httpd: running ok")

			go s.ListenAndServe()
			<-smgr.StopChan()

			return nil
		},
	})
}
