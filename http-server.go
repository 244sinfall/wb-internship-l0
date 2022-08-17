package main

import (
	"fmt"
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
			if r.URL.Query().Has("db") {
				dbMsg, err := p.getItemFromDb(orderUid)
				if err != nil {
					fmt.Println("Error parsing data from db: " + err.Error())
				}
				msg = dbMsg
			}
		}
		fp := path.Join("frontend", "item.html")
		tmpl, _ := template.ParseFiles(fp)
		err = tmpl.Execute(w, msg)
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte("Something went wrong on building a page"))
		}
	} else {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Use / to search items"))
	}
}
func (p *Persistence) showMainPage(w http.ResponseWriter, r *http.Request) {

	fp := path.Join("frontend", "index.html")
	tmpl, _ := template.ParseFiles(fp)
	err := tmpl.Execute(w, p.cache.messages)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Something went wrong on building a page"))
	}
}
