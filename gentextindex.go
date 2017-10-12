package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/olebedev/config"
	"io"
	"log"
	"os"
	"sort"
	"strings"
)

var (
	db  *sql.DB
	l   *log.Logger
	cfg *config.Config
)

const (
	logFileName    = "gentextindex.log"
	configFileName = "config.json"
)

func init() {

	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", logFileName, ":", err)
	}

	multi := io.MultiWriter(f, os.Stdout)
	l = log.New(multi, "main: ", log.Ldate|log.Ltime|log.Lshortfile)

	cfg, err := config.ParseJsonFile(configFileName)
	if err != nil {
		l.Fatalln("Failed to open config file", configFileName, ":", err)
	}

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

func GenerateSearchIndex(text string) string {
	text = strings.Replace(text, "/", " ", -1)
	text = strings.Replace(text, "[", " ", -1)
	text = strings.Replace(text, "(", " ", -1)
	text = strings.Replace(text, "]", " ", -1)
	text = strings.Replace(text, ")", " ", -1)
	text = strings.Replace(text, ".", " ", -1)
	text = strings.Replace(text, "_", " ", -1)
	uniq := map[string]int{}
	for _, s := range strings.Split(text, " ") {
		if s != "" {
			if cv, presen := uniq[s]; presen {
				uniq[s] = cv + 1
			} else {
				uniq[s] = 1
			}
		}
	}

	type kvt struct {
		Key   string
		Value int
	}

	kv := make([]kvt, len(uniq))

	for k, v := range uniq {
		kv = append(kv, kvt{k, v})
	}

	sort.Slice(kv, func(i, j int) bool { return kv[i].Value > kv[j].Value })

	text = ""

	for _, i := range kv {
		text = text + i.Key + " "
	}

	return strings.Trim(text, " ")

}

func main() {

	defer db.Close()

	var id int
	var name string
	var files bool
	var textIndex string

	infoHashSelect, err := db.Query("SELECT id, name, files FROM infohash WHERE textindex=\"\" LIMIT 10000")
	//infoHashSelect, err := db.Query("SELECT id, name, files FROM infohash WHERE textindex!=\"\" ")
	if err != nil {
		l.Panicln(err.Error())
	}

	for infoHashSelect.Next() {

		err := infoHashSelect.Scan(&id, &name, &files)
		if err != nil {
			l.Fatal(err)
		}

		l.Println(id, name)

		textIndex = name

		if files {

			filesSelect, err := db.Query("SELECT path FROM files WHERE infohash_id=? ", id)
			if err != nil {
				l.Panicln(err.Error())
			}

			for filesSelect.Next() {
				var path string
				err := filesSelect.Scan(&path)
				if err != nil {
					l.Fatal(err)
				}
				textIndex += " " + path
			}
		}
		l.Printf("Len of textIndex before %d", len(textIndex))
		textIndex = GenerateSearchIndex(textIndex)
		l.Printf("Len of textIndex after %d", len(textIndex))

		//l.Println(textIndex)

		upd, err := db.Prepare("UPDATE infohash SET textindex=? WHERE id=?")
		if err != nil {
			l.Panicln(err.Error())
		}

		_, err = upd.Exec(textIndex, id)
		if err != nil {
			l.Println(err.Error())
			continue
		}
		upd.Close()
	}

}
