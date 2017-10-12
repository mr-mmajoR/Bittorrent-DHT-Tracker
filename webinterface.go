package main

import (
	//"encoding/json"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	// "bufio"
	// "errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/olebedev/config"
	"html/template"
	"io"
	"log"
	// "net"
	"net/http"
	"os"
	"sort"
	"strings"
	// "sync"
	// "time"
)

var (
	db  *sql.DB
	l   *log.Logger
	cfg *config.Config

	bind_port      string
	bind_interface string

	main_tpl    *template.Template
	search_tpl  *template.Template
	details_tpl *template.Template
)

const (
	logFileName    = "webinterface.log"
	configFileName = "config.json"
	nl             = "\r\n"
)

type (
	file struct {
		Path   string
		Length string
	}

	Files []file

	bitTorrent struct {
		Id        int64
		InfoHash  string
		Name      string
		HaveFiles bool
		Files     []file
		Length    string
	}

	MainData struct {
		Title           string
		CountOfTorrents int
		Lastest         []bitTorrent
		Populatest      []bitTorrent
	}

	SearchData struct {
		Title   string
		Query   string
		Order   string
		Founded []bitTorrent
	}

	DetailData struct {
		Title   string
		Torrent bitTorrent
		Addeded string
		Updated string
	}
)

func (slice Files) Len() int {
	return len(slice)
}

func (slice Files) Less(i, j int) bool {
	return slice[i].Path < slice[j].Path
}

func (slice Files) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func init() {

	log.Println("Starting web interface")

	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file ", logFileName, ":", err)
	}

	multi := io.MultiWriter(f, os.Stdout)
	l = log.New(multi, "main: ", log.Ldate|log.Ltime|log.Lshortfile)

	cfg, err := config.ParseJsonFile(configFileName)
	if err != nil {
		l.Fatalln("Failed to parse config file ", configFileName, ":", err)
	}

	bind_port, _ = cfg.String("webinterface.port")
	bind_interface, _ = cfg.String("webinterface.interface")

	main_tpl = template.Must(template.New("main").ParseFiles("templates/base.html", "templates/main.html"))
	search_tpl = template.Must(template.New("search").ParseFiles("templates/base.html", "templates/search.html"))
	details_tpl = template.Must(template.New("details").ParseFiles("templates/base.html", "templates/details.html"))

	host, _ := cfg.String("database.host")
	name, _ := cfg.String("database.name")
	user, _ := cfg.String("database.user")
	pass, _ := cfg.String("database.password")

	db, err = sql.Open("mysql", ""+user+":"+pass+"@"+host+"/"+name)
	if err != nil {
		l.Fatalln("Error database connect", err.Error())
		panic(err.Error())
	}

}

func humanizeFileSize(length int) string {
	if length >= 1024*1024*1024 {
		return fmt.Sprintf("%d", (1.0*length)/(1024*1024*1024)) + " Gb"

	}
	if length >= 1024*1024 {
		return fmt.Sprintf("%d", (1.0*length)/(1024*1024)) + " Mb"

	}
	if length >= 1024 {
		return fmt.Sprintf("%d", length/1024) + "Kb"

	}
	return fmt.Sprintf("%d", length)
}

func getListOfTorrents(sqlQuery string) []bitTorrent {

	var (
		id       int64
		infohash string
		name     string
		length   int
		files    bool
	)

	res := []bitTorrent{}
	selQuery, err := db.Query(sqlQuery)
	if err != nil {
		l.Panicln(err.Error())
	}
	defer selQuery.Close()
	for selQuery.Next() {
		err := selQuery.Scan(&id, &infohash, &name, &length, &files)
		if err != nil {

			l.Fatal(err)
		}
		res = append(res, bitTorrent{
			Id:        id,
			InfoHash:  infohash,
			Name:      name,
			Length:    humanizeFileSize(length),
			HaveFiles: files,
		})
	}
	err = selQuery.Err()
	if err != nil {
		l.Fatal(err)
	}
	return res
}

func main_handler(w http.ResponseWriter, r *http.Request) {
	var (
		lastest         []bitTorrent
		populatest      []bitTorrent
		countOfTorrents int
	)

	err := db.QueryRow("SELECT count(id) FROM infohash").Scan(&countOfTorrents)
	if err != nil {
		l.Panicln(err.Error())
	}

	lastest = getListOfTorrents("SELECT id, infohash, name, length, files FROM infohash ORDER BY addeded DESC LIMIT 50")
	populatest = getListOfTorrents("SELECT id, infohash, name, length, files FROM infohash ORDER BY cnt DESC LIMIT 50")

	data := MainData{
		Title:           "Welcome to DHT search engine!",
		CountOfTorrents: countOfTorrents,
		Lastest:         lastest,
		Populatest:      populatest,
	}

	l.Printf("main page open. Count of torrents: %d", countOfTorrents)

	main_tpl.ExecuteTemplate(w, "base", data)

	return
}

func search_handler(w http.ResponseWriter, r *http.Request) {
	var (
		query   string
		order   string
		founded []bitTorrent
	)
	r.ParseForm()
	//fmt.Println(r.Form)

	query = r.FormValue("q")
	order = r.FormValue("order")

	l.Println("search by words [" + query + "] ")

	if order != "cnt" {
		order = "updated"
	}

	words := []string{}
	words2 := []string{}
	for _, w := range strings.Split(query, " ") {
		if w != "" {
			words = append(words, " `name` like \"%"+w+"%\" ")
			words2 = append(words2, " f.`path` like \"%"+w+"%\" ")
		}
	}
	condition := strings.Join(words, " and ")
	if len(condition) > 0 {
		condition = "WHERE " + condition
	}
	condition2 := strings.Join(words2, " and ")
	if len(condition2) > 0 {
		condition2 = "WHERE " + condition2
	}

	sql := "SELECT id, infohash, name, length, files FROM ("
	sql += "        SELECT id, infohash, name, length, files, updated, cnt FROM infohash " + condition
	sql += "    UNION"
	sql += "        SELECT DISTINCT f.infohash_id, ih.infohash, ih.name, ih.length, ih.files, ih.updated, ih.cnt FROM `files` as f LEFT JOIN infohash as ih on f.`infohash_id` = ih.id "
	sql += condition2
	sql += ") AS res ORDER BY res." + order + " DESC LIMIT 10000"

	//founded = getListOfTorrents("SELECT infohash, name, length, files FROM infohash " + condition + " ORDER BY " + order + " DESC LIMIT 10000")

	founded = getListOfTorrents(sql)

	l.Printf("\t founded [%d] results", len(founded))

	data := SearchData{
		Title:   "Result of search: " + query,
		Query:   query,
		Order:   order,
		Founded: founded,
	}
	search_tpl.ExecuteTemplate(w, "base", data)
}

func search2_handler(w http.ResponseWriter, r *http.Request) {
	var (
		query   string
		order   string
		founded []bitTorrent
	)
	r.ParseForm()
	//fmt.Println(r.Form)

	query = r.FormValue("q")
	order = r.FormValue("order")

	l.Println("search by words [" + query + "] ")

	if order != "cnt" {
		order = "updated"
	}

	words := []string{}
	for _, w := range strings.Split(query, " ") {
		if w != "" {
			words = append(words, " `textindex` like \"%"+w+"%\" ")
		}
	}
	condition := strings.Join(words, " and ")
	if len(condition) > 0 {
		condition = "WHERE " + condition
	}

	sql := "SELECT id, infohash, name, length, files FROM infohash " + condition
	sql += " ORDER BY " + order + " DESC LIMIT 10000"

	founded = getListOfTorrents(sql)

	l.Printf("\t founded [%d] results", len(founded))

	data := SearchData{
		Title:   "Result of search: " + query,
		Query:   query,
		Order:   order,
		Founded: founded,
	}
	search_tpl.ExecuteTemplate(w, "base", data)
}

func details_handler(w http.ResponseWriter, r *http.Request) {
	var (
		id        int64
		infohash  string
		name      string
		addeded   string
		updated   string
		length    int
		haveFiles bool
	)

	r.ParseForm()
	//fmt.Println(r.Form)

	hash_id := r.FormValue("id")
	row := db.QueryRow("SELECT id, infohash, name, length, files, addeded, updated FROM infohash WHERE id=" + hash_id)
	err := row.Scan(&id, &infohash, &name, &length, &haveFiles, &addeded, &updated)
	if err != nil {
		l.Panicln(err.Error())
	}

	l.Println("get details of [" + infohash + "] " + name + " (" + humanizeFileSize(length) + ")")

	files := Files{}
	if haveFiles {

		fs, err := db.Query("SELECT path, length FROM files WHERE infohash_id=" + fmt.Sprintf("%d", id))
		if err != nil {
			l.Panicln(err.Error())
		}
		defer fs.Close()

		var path string
		var filelength int

		for fs.Next() {

			err := fs.Scan(&path, &filelength)
			if err != nil {
				l.Fatal(err)
			}
			files = append(files, file{
				Path:   path,
				Length: humanizeFileSize(filelength),
			})
			l.Println("\t (" + humanizeFileSize(filelength) + ") " + path)

		}
		err = fs.Err()
		if err != nil {
			l.Fatal(err)
		}

	}

	sort.Sort(files)

	bt := bitTorrent{
		InfoHash:  infohash,
		Name:      name,
		HaveFiles: haveFiles,
		Files:     files,
		Length:    humanizeFileSize(length),
	}

	data := DetailData{
		Title:   "Details: " + bt.Name,
		Torrent: bt,
		Addeded: addeded,
		Updated: updated,
	}

	details_tpl.ExecuteTemplate(w, "base", data)
	// io.WriteString(w, sres)
}

func main() {

	rtr := mux.NewRouter()
	rtr.HandleFunc("/", main_handler).Methods("GET")
	rtr.HandleFunc("/search_old/", search_handler).Methods("GET")
	rtr.HandleFunc("/search/", search2_handler).Methods("GET")
	rtr.HandleFunc("/details/", details_handler).Methods("GET")
	//rtr.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	rtr.PathPrefix("/static/").Handler(s)
	http.Handle("/", rtr)

	l.Println("Listening on " + bind_interface + ":" + bind_port)

	http.ListenAndServe(bind_interface+":"+bind_port, nil) // set listen port

}
