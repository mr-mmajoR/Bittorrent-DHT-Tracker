package main

import (
	"database/sql"
	"encoding/hex"
	//"encoding/json"
	"flag"
	//"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/olebedev/config"
	"github.com/shiyanhui/dht"
	"io"
	"log"
	// "net/http"
	_ "net/http/pprof"
	"os"
	"sort"
	"strings"
	//"code.google.com/p/go.text/encoding/charmap"
	//"code.google.com/p/go.text/transform"
	//"runtime"
)

var (
	db  *sql.DB
	l   *log.Logger
	cfg *config.Config

	port string

	replacer *strings.Replacer
)

const (
	logFileName    = "logger.log"
	configFileName = "config.json"
)

type file struct {
	Path   []interface{} `json:"path"`
	Length int           `json:"length"`
}

type bitTorrent struct {
	InfoHash string `json:"infohash"`
	Name     string `json:"name"`
	Files    []file `json:"files,omitempty"`
	Length   int    `json:"length,omitempty"`
}

// func convertCP1251toUTF8(Text1251) {
// 	sr := strings.NewReader(Text1251)
// 	tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
// 	buf, err := ioutil.ReadAll(tr)
// 	if err != err {
// 		l.Println("Error string converting")
// 	}
// 	return string(buf)
// }

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

	port, _ = cfg.String("spider.port")

	if port == "" {
		port = "6882"
	}

	db, err = sql.Open("mysql", ""+user+":"+pass+"@"+host+"/"+name)
	if err != nil {
		l.Fatalln("Error database connect", err.Error())
		panic(err.Error())
	}

	portPtr := flag.String("port", port, "DHT port")
	flag.Parse()
	port = *portPtr

	replacer = strings.NewReplacer(
		"/", " ",
		"[", " ",
		"(", " ",
		"]", " ",
		")", " ",
		".", " ",
		"_", " ",
	)

}

func GenerateSearchIndex(text string) string {
	// text = strings.Replace(text, "/", " ", -1)
	// text = strings.Replace(text, "[", " ", -1)
	// text = strings.Replace(text, "(", " ", -1)
	// text = strings.Replace(text, "]", " ", -1)
	// text = strings.Replace(text, ")", " ", -1)
	// text = strings.Replace(text, ".", " ", -1)
	// text = strings.Replace(text, "_", " ", -1)
	text = replacer.Replace(text)
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

	w := dht.NewWire(65536, 1024, 256)
	textIndex := ""
	countDown := 100
	go func() {
		for resp := range w.Response() {
			metadata, err := dht.Decode(resp.MetadataInfo)
			if err != nil {
				continue
			}
			info := metadata.(map[string]interface{})

			if _, ok := info["name"]; !ok {
				continue
			}

			bt := bitTorrent{
				InfoHash: hex.EncodeToString(resp.InfoHash),
				Name:     info["name"].(string),
			}

			var vFiles []interface{}
			haveFiles := false
			isNew := false

			if v, ok := info["files"]; ok {
				haveFiles = true
				vFiles = v.([]interface{})
				bt.Files = make([]file, len(vFiles))

				for i, item := range vFiles {
					f := item.(map[string]interface{})
					bt.Files[i] = file{
						Path:   f["path"].([]interface{}),
						Length: f["length"].(int),
					}
				}
			} else if _, ok := info["length"]; ok {
				bt.Length = info["length"].(int)
			}

			// data, err := json.Marshal(bt)
			// if err == nil {
			//l.Printf("[%s]  %s \t%d\n", bt.InfoHash, bt.Name, bt.Length)
			infoHashSelect, err := db.Query("SELECT id FROM infohash WHERE infohash = ?", bt.InfoHash)
			if err != nil {
				l.Panicln(err.Error())
			}

			var id int64

			if infoHashSelect.Next() {

				err := infoHashSelect.Scan(&id)
				if err != nil {
					l.Fatal(err)
				}

				l.Println(bt.InfoHash, "   update: \t"+bt.Name)

				//upd, err := db.Prepare("UPDATE infohash SET name=?,files=?,length=?,updated=NOW(), cnt=cnt+1 WHERE infohash=?")
				upd, err := db.Prepare("UPDATE infohash SET updated=NOW(), cnt=cnt+1 WHERE id=?")
				if err != nil {
					l.Panicln(err.Error())
				}

				_, err = upd.Exec(id)
				if err != nil {
					l.Println(err.Error())
					continue
				}
				upd.Close()

			} else {

				isNew = true

				totalLength := 0
				textIndex = bt.Name

				if _, ok := info["length"]; ok {
					totalLength = info["length"].(int)
				}

				if haveFiles {

					for _, item := range vFiles {
						f := item.(map[string]interface{})
						totalLength += f["length"].(int)
						for _, p := range f["path"].([]interface{}) {
							textIndex += " " + p.(string)
						}

					}
				}

				textIndex = GenerateSearchIndex(textIndex)

				l.Println(bt.InfoHash, "   add new:\t"+bt.Name)

				ins, err := db.Prepare("INSERT INTO infohash SET infohash=?,name=?,files=?,length=?,addeded=NOW(),updated=NOW(), textindex=?")
				if err != nil {
					l.Panicln(err.Error())
				}

				res, err := ins.Exec(bt.InfoHash, bt.Name, haveFiles, totalLength, textIndex)
				if err != nil {
					l.Println(err.Error())
					continue
				}

				id, err = res.LastInsertId()
				if err != nil {
					println("Error:", err.Error())
				}

				ins.Close()
			}

			infoHashSelect.Close()

			textIndex = ""

			if haveFiles && isNew {

				ins, err := db.Prepare("INSERT INTO files SET infohash_id=?,path=?,length=?")
				if err != nil {
					l.Panicln(err.Error())
				}

				for _, item := range vFiles {
					f := item.(map[string]interface{})
					path := ""
					for _, p := range f["path"].([]interface{}) {
						if path != "" {
							path = path + "/" + p.(string)
						} else {
							path = p.(string)
						}
					}
					_, err = ins.Exec(id, path, f["length"].(int))
					if err != nil {
						l.Println(err.Error())
					}

				}

				ins.Close()

				l.Printf("%s    files:\t%d", bt.InfoHash, len(vFiles))

			}

			countDown--
			if countDown <= 0 {
				// exit to restart in run.sh
				//l.Panic("Exit to restart")
				os.Exit(0)
			}

		}
	}()

	go w.Run()

	config := dht.NewCrawlConfig()
	l.Println("Use port ", port)

	config.Address = ":" + port
	config.PrimeNodes = append(config.PrimeNodes, "router.bitcomet.com:6881")

	config.OnAnnouncePeer = func(infoHash, ip string, port int) {
		//l.Println("Annonce peer", ip, ":", port)
		w.Request([]byte(infoHash), ip, port)
	}
	d := dht.New(config)

	d.Run()

	defer db.Close()
}
