package main

import (
	dbpkg "candidate_alocator/back/db"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

// servePreview exposes a development-only, loopback-by-default adapter to the
// real Wails App methods, for review on hosts without native GTK/WebKit.
func servePreview(args []string) error {
	f := flag.NewFlagSet("preview", flag.ContinueOnError)
	listen := f.String("listen", "127.0.0.1:8128", "endereço do preview de desenvolvimento")
	if err := f.Parse(args); err != nil {
		return err
	}
	a := NewApp()
	defer a.clearTransientState()
	if err := dbpkg.EnsureDefaultDatabase(); err != nil {
		return err
	}
	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return err
	}
	router := previewRouter(a, dist)
	if err := a.startOperations(); err != nil {
		return err
	}
	server := &http.Server{Addr: *listen, Handler: router, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: time.Minute}
	fmt.Printf("Preview de desenvolvimento: http://%s (base mainjoao)\n", *listen)
	return server.ListenAndServe()
}

func previewRouter(a *App, dist fs.FS) http.Handler {
	methods := []string{"SuggestMapping", "SuggestMappingAvaliador", "SuggestMappingRestricao", "GetCandidateMappingFieldInfos", "GetAvaliadorMappingFieldInfos", "GetRestricaoMappingFieldInfos", "BuildUsuariosWithMapping", "BuildAvaliadoresWithMapping", "BuildRestricoesWithMapping", "SaveRestricoesFromMaps", "SaveUsuariosFromMaps", "SaveAvaliadoresFromMaps", "GetWorkflowDefinition", "GetCriteriaOptions", "BuildAllocationConfigurationFromDatabase", "GetPreferenceScheduleMappings", "StartAllocation", "StopAllocation", "GetAllocationRunState", "GetAgenda", "EvaluateCandidates", "GetMoveSuggestions", "MoveAgenda", "GetChannelConfig", "SaveChannelConfig", "TestChannels", "PreviewAgendaNotices", "SendAgendaNotices", "GetNoticeHistory", "GetNoticeDetails", "RetryNoticeFailures", "GetNoticeStates", "ExportAgendaExcel", "GetGoogleConnectionStatus", "BeginGoogleConnection", "DisconnectGoogle"}
	allowed := map[string]bool{}
	for _, m := range methods {
		allowed[m] = true
	}
	mux := http.NewServeMux()
	var mu sync.Mutex
	listeners := map[chan int64]bool{}
	a.noticeEvent = func(id int64) {
		mu.Lock()
		defer mu.Unlock()
		for ch := range listeners {
			select {
			case ch <- id:
			default:
			}
		}
	}
	mux.HandleFunc("GET /preview/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		ch := make(chan int64, 1)
		mu.Lock()
		listeners[ch] = true
		mu.Unlock()
		defer func() { mu.Lock(); delete(listeners, ch); mu.Unlock() }()
		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()
		for {
			select {
			case id := <-ch:
				_, _ = fmt.Fprintf(w, "data: %d\n\n", id)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	mux.HandleFunc("POST /preview/call/{method}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		method := r.PathValue("method")
		if !allowed[method] {
			http.Error(w, `{"error":"método não disponível"}`, 404)
			return
		}
		callPreviewMethod(a, method, w, r)
	})
	bootstrap := `const handlers=new Map();window.runtime={EventsOnMultiple:(n,fn)=>{if(!handlers.has(n))handlers.set(n,new Set());handlers.get(n).add(fn);return ()=>handlers.get(n)?.delete(fn)},EventsOff:(n)=>handlers.delete(n),EventsEmit:(n,...args)=>handlers.get(n)?.forEach(fn=>fn(...args)),BrowserOpenURL:(u)=>window.open(u,'_blank','noopener'),LogPrint:()=>{}};window.go={main:{App:{}}};for(const name of METHODS){window.go.main.App[name]=async(...args)=>{const r=await fetch('/preview/call/'+name,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(args)});const data=await r.json();if(!r.ok)throw new Error(data.error);return data.value}}const events=new EventSource('/preview/events');events.onmessage=()=>window.runtime.EventsEmit('notices:changed');`
	names, _ := json.Marshal(methods)
	bootstrap = strings.Replace(bootstrap, "METHODS", string(names), 1)
	mux.HandleFunc("GET /preview/bootstrap.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = fmt.Fprint(w, bootstrap)
	})
	files := http.FileServer(http.FS(dist))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" && path != "index.html" {
			if file, err := dist.Open(path); err == nil {
				_ = file.Close()
				files.ServeHTTP(w, r)
				return
			}
		}
		data, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "frontend indisponível", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, strings.Replace(string(data), "<head>", `<head><script src="/preview/bootstrap.js"></script>`, 1))
	})
	return mux
}

func callPreviewMethod(a *App, name string, w http.ResponseWriter, r *http.Request) {
	var args []json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(&args); err != nil {
		http.Error(w, `{"error":"pedido inválido"}`, 400)
		return
	}
	method := reflect.ValueOf(a).MethodByName(name)
	if !method.IsValid() || len(args) != method.Type().NumIn() {
		http.Error(w, `{"error":"argumentos inválidos"}`, 400)
		return
	}
	values := make([]reflect.Value, len(args))
	for i, arg := range args {
		value := reflect.New(method.Type().In(i))
		if err := json.Unmarshal(arg, value.Interface()); err != nil {
			http.Error(w, `{"error":"argumento inválido"}`, 400)
			return
		}
		values[i] = value.Elem()
	}
	results := method.Call(values)
	var value any
	for _, v := range results {
		if v.Type().Implements(reflect.TypeFor[error]()) {
			if !v.IsNil() {
				w.WriteHeader(400)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": v.Interface().(error).Error()})
				return
			}
		} else {
			value = v.Interface()
		}
	}
	if len(results) == 0 {
		value = true
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
}
