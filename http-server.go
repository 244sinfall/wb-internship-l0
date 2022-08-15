package main

import (
	"html/template"
	"net/http"
	"path"
	"strings"
)

func (p *Persistence) getItem(w http.ResponseWriter, r *http.Request) {
	orderUid := strings.TrimPrefix(r.URL.Path, "/orders/")
	if orderUid != "" {
		msg, err := p.cache.get(orderUid)
		if err != nil {

		}
		fp := path.Join("frontend", "item.html")
		tmpl, _ := template.ParseFiles(fp)
		tmpl.Execute(w, msg)
	} else {
		w.WriteHeader(400)
		w.Write([]byte("Use / to search items"))
	}
}
func (p *Persistence) showMainPage(w http.ResponseWriter, r *http.Request) {
	fp := path.Join("frontend", "index.html")
	tmpl, _ := template.ParseFiles(fp)
	tmpl.Execute(w, p.cache.messages)
}

//func getItem(w http.ResponseWriter, r *http.Request) {
//	fmt.Printf("got /hello request\n")
//	_, err := io.WriteString(w, "Hello, HTTP!\n")
//	if err != nil {
//		return
//	}
//}
