package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const (
	confFile = ".godbtool.json"
	fileMod  = 0644
)

var (
	globalMap map[string]*mysqlConfig
)

type dbConfig struct {
	Name      string      `json:"name"`
	MysqlConf mysqlConfig `json:"mysql_conf"`
}
type mysqlConfig struct {
	Alias string `json:"-"`
	Host  string `json:"host"`
	Port  string `json:"port"`
	User  string `json:"user"`
	Pwd   string `json:"pwd"`
}

func initConf() error {
	u, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	u = path.Join(u, confFile)
	_, err = os.Stat(u)
	if os.IsNotExist(err) {
		f, err := os.OpenFile(u, os.O_CREATE|os.O_RDWR, fileMod)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("{}"))
		if err != nil {
			return err
		}
		defer f.Close()
	}

	b, err := ioutil.ReadFile(u)
	if err != nil {
		return err
	}
	globalMap = map[string]*mysqlConfig{}
	return json.Unmarshal(b, &globalMap)
}
func main() {
	err := initConf()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	app := cli.App{
		Name:        "godbtool",
		Usage:       "db tool",
		Description: "table to struct,struct to table",
		Commands:    commands(),
	}
	err = app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

//addDbConn godbtool add local 127.0.0.1 3306 root 12345678
func addDbConn(ctx *cli.Context) error {
	if ctx.NArg() != 5 {
		return errors.New("godbtool add args invalid example:godbtool add local 127.0.0.1 3306 root 12345678")
	}
	name := ctx.Args().Get(0)
	host := ctx.Args().Get(1)
	port := ctx.Args().Get(2)
	user := ctx.Args().Get(3)
	pwd := ctx.Args().Get(4)

	globalMap[name] = &mysqlConfig{
		Host: host,
		Port: port,
		User: user,
		Pwd:  pwd,
	}
	return modifyConf()
}
func modifyConf() error {
	raw, err := json.Marshal(globalMap)
	if err != nil {
		return err
	}
	u, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	u = path.Join(u, confFile)
	err = ioutil.WriteFile(u, raw, 0644)
	if err != nil {
		return err
	}
	return err
}
func delDbConn(ctx *cli.Context) error {
	name := ctx.Args().Get(0)
	if _, ok := globalMap[name]; !ok {
		return fmt.Errorf("not found %s", name)
	}
	delete(globalMap, name)
	return modifyConf()
}
func toStruct(ctx *cli.Context) error {
	if ctx.NArg() != 3 {
		return errors.New("godbtool tostruct args invalid example:godbtool tostruct local table model.go")
	}
	name := ctx.Args().Get(0)
	tName := ctx.Args().Get(1)
	dst := ctx.Args().Get(2)
	conf, ok := globalMap[name]
	if !ok {
		return fmt.Errorf("not found :%s,see godbtool add", name)
	}
	dbTable := strings.Split(tName, ".")
	if len(dbTable) != 2 {
		return fmt.Errorf("dbtable invalid example demo.user")
	}
	t2s := NewTable2Struct()
	err := t2s.
		SavePath(dst).
		Dsn(&DsnConf{
			Ip:       conf.Host,
			Port:     conf.Port,
			DataBase: dbTable[0],
			User:     conf.User,
			Pwd:      conf.Pwd,
		}).
		Table(dbTable[1]).DateToTime(true).TagKey("gorm").EnableJsonTag().PackageName("main").Config(&T2tConfig{
		SeperatFile:      true,
		StructNameToHump: true,
	}).Run()
	return err
}

//godbtool totable local model.go
func toTable(ctx *cli.Context) error {
	name := ctx.Args().Get(0)
	file := ctx.Args().Get(1)
	conf, ok := globalMap[name]
	if !ok {
		return fmt.Errorf("not found :%s,see godbtool add", name)
	}
	//re()
	return struct2Table(file, &DsnConf{
		Ip:   conf.Host,
		Port: conf.Port,
		User: conf.User,
		Pwd:  conf.Pwd,
	})
}
func commands() cli.Commands {
	tmp := cli.Commands{
		&cli.Command{
			Name:   "add",
			Usage:  "add db connect,example:godbtool add local 127.0.0.1 3306 root 12345678",
			Action: addDbConn,
		},
		&cli.Command{
			Name:   "del",
			Usage:  "del db connect,example:godbtool del local",
			Action: delDbConn,
		},
		&cli.Command{
			Name:   "tostruct",
			Usage:  "table to struct,example:godbtool tostruct local table model.go",
			Action: toStruct,
		},
		&cli.Command{
			Name:   "totable",
			Usage:  "struct to table,example:godbtool totable local model.go",
			Action: toTable,
		},
		&cli.Command{
			Name:   "get",
			Usage:  "display conf,example:godbtool get [optional]",
			Action: displayConf,
		},
		&cli.Command{
			Name:   "todjango",
			Usage:  "table to django code,example:godbtool todjango local table",
			Action: toDjango,
		},
	}
	return tmp
}

func displayConf(ctx *cli.Context) error {
	name := ""
	if ctx.NArg() == 1 {
		name = ctx.Args().Get(0)
	}
	if ctx.NArg() > 1 {
		return errors.New("invalid args")
	}
	list := make([]*mysqlConfig, 0)
	for k, v := range globalMap {
		v := v
		v.Alias = k
		if name != "" {
			if strings.Contains(k, name) {
				list = append(list, v)
			}
		} else {
			list = append(list, v)
		}
	}
	fmt.Println(Table(list))
	return nil
}

//godbtool todjango local table
func toDjango(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("godbtool tostruct args invalid example:godbtool todjango local user.user_profile")
	}
	name := ctx.Args().Get(0)
	tName := ctx.Args().Get(1)

	conf, ok := globalMap[name]
	if !ok {
		return fmt.Errorf("not found :%s,see godbtool add", name)
	}
	dbTable := strings.Split(tName, ".")
	if len(dbTable) != 2 {
		return fmt.Errorf("dbtable invalid example demo.user")
	}
	t2d := NewTable2Django()
	err := t2d.Dsn(&DsnConf{
		Ip:       conf.Host,
		Port:     conf.Port,
		DataBase: dbTable[0],
		User:     conf.User,
		Pwd:      conf.Pwd,
	}).Table(dbTable[1]).Run()
	return err
}
