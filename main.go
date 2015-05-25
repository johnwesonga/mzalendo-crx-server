package mzalendo

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/johnwesonga/go-mzalendo/mzalendo"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/person", personHandler)
	http.HandleFunc("/organization", organizationHandler)
}

type PersonResponse struct {
	Name        string
	Image       string
	Memberships []string
	PersonId    string
	BirthDate   string
}

func createHttpClient(c context.Context) *http.Client {
	deadline := time.Duration(60) * time.Second
	client := &http.Client{
		Transport: &urlfetch.Transport{
			Context:  c,
			Deadline: deadline,
		},
	}

	return client

}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func homeHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Mzalendo Service")
}

func personHandler(w http.ResponseWriter, req *http.Request) {
	defer timeTrack(time.Now(), "personHandler")

	orgSlice := make([]string, 0)
	context := appengine.NewContext(req)
	client := createHttpClient(context)
	var newItems []*memcache.Item
	c := mzalendo.NewClient(client)

	if id := req.FormValue("id"); id != "" {
		//check if item exists in cache

		if getFromCache(context, "core_person:"+id) {
			item, err := memcache.Get(context, "core_person:"+id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("item found in cache %v", string(item.Value))
			fmt.Fprint(w, string(item.Value))
			return
		}

		r, err := c.Api.GetPerson(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		counter := 0
		for _, v := range r.Result.Memberships {
			if counter >= 3 {
				break
			}
			orgParts := strings.Split(v.OrganizationID, ":")

			if len(orgParts) > 1 {
				//check if item exists in cache
				orgId := orgParts[1]
				if !getFromCache(context, "core_organization:"+orgId) {

					org, err := c.Api.GetOrganization(orgParts[1])
					if err != nil {

						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					affiliation := fmt.Sprintf(" %v of %v", v.Role, org.Result.Name)

					newItems = append(newItems, &memcache.Item{Key: "core_organization:" + orgId, Value: []byte(affiliation), Expiration: (time.Duration(60) * time.Minute)})
					memcache.SetMulti(context, newItems)

					orgSlice = append(orgSlice, affiliation)

				} else {
					item, err := memcache.Get(context, "core_organization:"+orgId)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					orgSlice = append(orgSlice, string(item.Value))
				}

			}
			counter++
		}

		p := PersonResponse{Name: r.Result.Name,
			Image:       r.Result.Images[0].ProxyURL,
			Memberships: orgSlice,
			PersonId:    r.Result.ID,
			BirthDate:   r.Result.BirthDate,
		}

		t := template.Must(template.New("").Parse(`<div id='my-tooltip-2986234'><div style='min-width:200px !important;'><a target="_blank" href="https://kenyan-politicians.popit.mysociety.org/persons/core_person:{{.PersonId}}"><h1>{{.Name}}</h1></a><img src="{{.Image}}/128/0"/></div><table><tr><td><b>BORN:</b>{{.BirthDate}}</td></tr><tr><td><b>PARTIES & AFFILIATIONS</b> </td></tr><tr><td>{{range .Memberships}}<tr><td>{{.}}</td></tr>{{end}}<td></td></table><h2>data by <a target="_blank" href="https://kenyan-politicians.popit.mysociety.org/persons/core_person:{{.PersonId}}">Mzalendo</a><h2></div>`))

		if err := t.Execute(w, &p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var tmplBuffer bytes.Buffer
		t.Execute(&tmplBuffer, &p)
		tmplCache := tmplBuffer.String()

		//cache the output
		memcache.Set(context, &memcache.Item{Key: "core_person:" + id, Value: []byte(tmplCache), Expiration: (time.Duration(60) * time.Minute)})

	}

}

func organizationHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "org")
}
