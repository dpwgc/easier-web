package easierweb

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/websocket"
	"net/http"
)

type RouterOptions struct {
	RootPath               string
	MultipartFormMaxMemory int64
	ErrorHandle            ErrorHandle
	RequestHandle          RequestHandle
	ResponseHandle         ResponseHandle
	CloseConsolePrint      bool
}

type Router struct {
	rootPath               string
	multipartFormMaxMemory int64
	router                 *httprouter.Router
	server                 http.Server
	middlewares            []Handle
	errorHandle            ErrorHandle
	requestHandle          RequestHandle
	responseHandle         ResponseHandle
	closeConsolePrint      bool
}

type PluginOptions struct {
	RequestHandle  RequestHandle
	ResponseHandle ResponseHandle
}

type Handle func(ctx *Context)

type RequestHandle func(ctx *Context, reqObj any) error

type ResponseHandle func(ctx *Context, result any, err error)

type ErrorHandle func(ctx *Context, err any)

func New(opts ...RouterOptions) *Router {
	r := &Router{
		multipartFormMaxMemory: 32 << 20,
		router:                 httprouter.New(),
		errorHandle:            defaultErrorHandle,
		requestHandle:          defaultRequestHandle,
		responseHandle:         defaultResponseHandle,
	}
	for _, v := range opts {
		if v.RootPath != "" {
			r.rootPath = v.RootPath
		}
		if v.MultipartFormMaxMemory > 0 {
			r.multipartFormMaxMemory = v.MultipartFormMaxMemory
		}
		if v.ErrorHandle != nil {
			r.errorHandle = v.ErrorHandle
		}
		if v.RequestHandle != nil {
			r.requestHandle = v.RequestHandle
		}
		if v.ResponseHandle != nil {
			r.responseHandle = v.ResponseHandle
		}
		r.closeConsolePrint = v.CloseConsolePrint
	}
	return r
}

// easier usage function

func (r *Router) EasyGET(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.GET(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyHEAD(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.HEAD(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyOPTIONS(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.OPTIONS(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyPOST(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.POST(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyPUT(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.PUT(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyPATCH(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.PATCH(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyDELETE(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.DELETE(path, r.buildHandle(easyHandle, opts))
}

func (r *Router) EasyAny(path string, easyHandle any, opts ...PluginOptions) *Router {
	return r.Any(path, r.buildHandle(easyHandle, opts))
}

// basic usage function

func (r *Router) GET(path string, handle Handle) *Router {
	r.router.GET(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) HEAD(path string, handle Handle) *Router {
	r.router.HEAD(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) OPTIONS(path string, handle Handle) *Router {
	r.router.OPTIONS(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) POST(path string, handle Handle) *Router {
	r.router.POST(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) PUT(path string, handle Handle) *Router {
	r.router.PUT(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) PATCH(path string, handle Handle) *Router {
	r.router.PATCH(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) DELETE(path string, handle Handle) *Router {
	r.router.DELETE(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		r.handle(handle, res, req, par, nil)
	})
	return r
}

func (r *Router) Any(path string, handle Handle) *Router {
	r.GET(path, handle)
	r.HEAD(path, handle)
	r.OPTIONS(path, handle)
	r.POST(path, handle)
	r.PUT(path, handle)
	r.PATCH(path, handle)
	r.DELETE(path, handle)
	return r
}

func (r *Router) WS(path string, handle Handle) *Router {
	r.router.GET(r.rootPath+path, func(res http.ResponseWriter, req *http.Request, par httprouter.Params) {
		websocket.Server{
			Handler: func(ws *websocket.Conn) {
				r.handle(handle, res, req, par, ws)
			},
			Handshake: func(config *websocket.Config, req *http.Request) error {
				// 解决跨域
				return nil
			},
		}.ServeHTTP(res, req)
	})
	return r
}

func (r *Router) Static(path, dir string) *Router {
	return r.StaticFS(path, http.Dir(dir))
}

func (r *Router) StaticFS(path string, fs http.FileSystem) *Router {
	r.router.ServeFiles(r.rootPath+path, fs)
	return r
}

func (r *Router) Use(middlewares ...Handle) *Router {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

func (r *Router) Run(addr string) error {
	r.consoleStartPrint(addr)
	r.server = http.Server{
		Addr:    addr,
		Handler: r.router,
	}
	return r.server.ListenAndServe()
}

func (r *Router) RunTLS(addr string, certFile string, keyFile string, tlsConfig *tls.Config) error {
	r.consoleStartPrint(addr)
	r.server = http.Server{
		Addr:      addr,
		Handler:   r.router,
		TLSConfig: tlsConfig,
	}
	return r.server.ListenAndServeTLS(certFile, keyFile)
}

func (r *Router) Close() error {
	return r.server.Shutdown(context.Background())
}

func (r *Router) consoleStartPrint(addr string) {
	if r.closeConsolePrint {
		return
	}
	fmt.Println("  ______          _        __          __  _     \n |  ____|        (_)       \\ \\        / / | |    \n | |__   __ _ ___ _  ___ _ _\\ \\  /\\  / /__| |__  \n |  __| / _` / __| |/ _ \\ '__\\ \\/  \\/ / _ \\ '_ \\ \n | |___| (_| \\__ \\ |  __/ |   \\  /\\  /  __/ |_) |\n |______\\__,_|___/_|\\___|_|    \\/  \\/ \\___|_.__/")
	fmt.Printf("\033[1;32;40m%s\033[0m\n", fmt.Sprintf(" >>> http server runs on [%s] ", addr))
}
