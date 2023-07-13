package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"plugin"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	baseSrc = `
package %s
import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)
type structFiled struct {
	Name       string
	Type       string
	Tag        string
	Annotation string
}
type Model struct {
	TableName string
	Struct    string
	DataBase  string
	Fileds    []structFiled
}

func GenStruct() []byte {
	result := make([]Model, 0)
	ptrs := newModels()
	for _, p := range ptrs {
		vl := reflect.ValueOf(p)
		f := vl.MethodByName("TableName")
		tf := f.Interface().(func() string)
		tableName := tf()
		vt := reflect.TypeOf(p)
		vt = vt.Elem()
		fileds := make([]structFiled, 0, vt.NumField())
		for i := 0; i < vt.NumField(); i++ {
			tag := vt.Field(i).Tag.Get("gorm")
			t:=""
			if strings.Contains(tag, "column"){
				tagF := strings.Split(tag, ":")
				if len(tagF) == 2 {
					t = tagF[1]
				}
			}
			fileds = append(fileds, structFiled{
				Name:       vt.Field(i).Name,
				Type:       vt.Field(i).Type.String(),
				Tag:        t,
				Annotation: "",
			})
		}
		dataBaseInfo := strings.Split(tableName, ".")
		if len(dataBaseInfo) < 2 {
			panic(errors.New("invalid TableName"))
		}
		l := strings.Split(vt.String(), ".")
		structName := l[len(l)-1]
		result = append(result, Model{
			TableName: dataBaseInfo[1],
			Struct:    structName,
			DataBase:  dataBaseInfo[0],
			Fileds:    fileds,
		})
	}
	raw, _ := json.Marshal(result)
	return raw
}

func newModels() []interface{} {
	return []interface{}{%s}
}
`
)

func exeSql(m exeModel, dsnConf *DsnConf) error {
	dsnConf.DataBase = m.DataBase
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", dsnConf.User, dsnConf.Pwd, dsnConf.Ip, dsnConf.Port, dsnConf.DataBase)
	Db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	r, err := Db.Exec(m.SqlSrc)
	if err != nil {
		return err
	}
	fmt.Println(r.RowsAffected())
	return nil
}
func struct2TableV2(srcFile string, conf *mysqlConfig) error {
	srcRaw, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return err
	}
	structs, annotationMap, pkg := exportStruct(BytesToString(srcRaw))
	if len(structs) == 0 {
		return fmt.Errorf("src invalid,file:%s", srcFile)
	}
	newFile, err := writeTmpSrcFile(structs, pkg)
	if err != nil {
		return err
	}
	raw, err := loadSOFile(newFile, srcFile)
	if err != nil {
		return err
	}
	mList := make([]Model, 0)
	err = json.Unmarshal(raw, &mList)
	if err != nil {
		return err
	}
	m := make(map[string]Model)
	for i := 0; i < len(mList); i++ {
		model := mList[i]
		am := annotationMap[model.Struct]
		for j := 0; j < len(model.Fileds); j++ {
			a := am[model.Fileds[j].Name]
			model.Fileds[j].Annotation = a
			if model.Fileds[j].Tag == "" {
				model.Fileds[j].Tag = humpToFiledRule(model.Fileds[j].Name)
			}
		}
		m[model.Struct] = model
	}
	sqlList := genTableSql(m)
	for _, v := range sqlList {
		err = exeSql(v, &DsnConf{
			Ip:       conf.Host,
			Port:     conf.Port,
			DataBase: v.DataBase,
			User:     conf.User,
			Pwd:      conf.Pwd,
		})
		if err != nil {
			return err
		}
	}
	return nil
	//db, err := connMysql(conf)
	//if err != nil {
	//	return err
	//}
	//return db.Migrator().AutoMigrate(new(GoadminOperationLog))
}

func loadSOFile(file string, src string) ([]byte, error) {
	soFile := fmt.Sprintf("%s.so", file)
	cmd := exec.Command(
		"go", "build",
		"-buildmode", "plugin",
		"-o", soFile,
		file, src,
	)
	defer func() {
		os.Remove(file)
	}()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	defer func() {
		os.Remove(soFile)
	}()
	p, err := plugin.Open(soFile)
	if err != nil {
		return nil, err
	}
	s, err := p.Lookup("GenStruct")
	if err != nil {
		return nil, err
	}
	ms := s.(func() []byte)()
	return ms, nil
}
func writeTmpSrcFile(structs []string, pkgName string) (string, error) {
	newCode := genSrc(structs, pkgName)
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(newCode)
	newName := fmt.Sprintf("./tmp%d.go", time.Now().Unix())
	err := ioutil.WriteFile(newName, buffer.Bytes(), 0644)
	if err != nil {
		return "", err
	}
	return newName, nil
}

func genSrc(structs []string, pkgName string) string {
	list := make([]string, 0, len(structs))
	for _, v := range structs {
		list = append(list, fmt.Sprintf("new(%s)", v))
	}
	s := strings.Join(list, ",")
	return fmt.Sprintf(baseSrc, pkgName, s)
}
func exportStruct(src string) ([]string, map[string]map[string]string, string) {
	r := regexp.MustCompile(`( )+|(\n)+`)
	src = r.ReplaceAllString(src, "$1$2")
	list := strings.Split(src, "\n")
	rsp := make([]string, 0)
	idx := 0
	m := make(map[string]map[string]string)
	pkgName := ""
	for idx < len(list) {
		line := list[idx]
		if strings.Contains(line, "package") {
			tmp := strings.Split(line, " ")
			if len(tmp) < 2 {
				panic(errors.New("invalid src"))
			}
			pkgName = tmp[1]
		}
		if strings.Contains(line, "type") && strings.Contains(line, "struct") {
			tmp := strings.Split(line, " ")
			if len(tmp) < 2 {
				continue
			}
			structName := tmp[1]
			rsp = append(rsp, structName)
			fMap, ok := m[structName]
			if !ok {
				fMap = make(map[string]string)
			}
			for i := idx + 1; i < len(list); i++ {
				if list[i] == "}" {
					line = list[i]
					idx = i
					break
				}
				list[i] = strings.ReplaceAll(list[i], "\t", "")
				list[i] = strings.TrimSpace(list[i])

				fileds := strings.Split(list[i], " ")
				if len(fileds) < 2 {
					continue
				}

				name := fileds[0]
				a := exportAnnotation(list[i])
				if a != "" {
					fMap[name] = a
				}
			}
			m[structName] = fMap
		}
		idx++
	}
	return rsp, m, pkgName
}

func humpToFiledRule(s string) string {
	newS := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if unicode.IsUpper(rune(s[i])) && i == 0 {
			newS.WriteString(strings.ToLower(string(s[i])))
		} else if unicode.IsUpper(rune(s[i])) {
			newS.WriteString(strings.ToLower("_" + string(s[i])))
		} else {
			newS.WriteByte(s[i])
		}
	}
	return newS.String()
}
