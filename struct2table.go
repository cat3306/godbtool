package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode"
	"unsafe"
)

const (
	Tag        = "column"
	Annotation = "//"
)

//CREATE TABLE `node_router_info` (
//`id` int NOT NULL AUTO_INCREMENT,
//`router_mac` varchar(128) NOT NULL DEFAULT '',
//`router_ip` varchar(32) NOT NULL DEFAULT '',
//`fake_router_ip` varchar(32) NOT NULL DEFAULT '',
//`node_ip` varchar(32) NOT NULL DEFAULT '',
//`fake_node_ip` varchar(32) NOT NULL DEFAULT '',
//`node_id` int NOT NULL DEFAULT '0',
//`node_host` varchar(128) NOT NULL DEFAULT '',
//`province` varchar(128) NOT NULL DEFAULT '',
//`port` int NOT NULL DEFAULT '0',
//`create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
//`update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
//`operator` varchar(128) NOT NULL DEFAULT '',
//`province_id` int NOT NULL DEFAULT '0',
//`op_code` int NOT NULL DEFAULT '2' COMMENT '1:重启 2:闲置状态',
//PRIMARY KEY (`id`),
//UNIQUE KEY `indx_mac` (`router_mac`,`fake_router_ip`,`fake_node_ip`,`port`) USING BTREE,
//KEY `idx_node_id` (`node_id`,`fake_router_ip`,`fake_node_ip`,`port`) USING BTREE
//) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
var typeForGoToMysql = map[string]string{
	"id":    "`%s` int NOT NULL AUTO_INCREMENT COMMENT '%s',\n",
	"int":   "`%s` int NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"int64": "`%s` bigint NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"int32": "`%s` int NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"int16": "`%s` smallint NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"int8":  "`%s` tinyint NOT NULL DEFAULT '0' COMMENT '%s',\n",

	"uint":   "`%s` int UNSIGNED NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"uint64": "`%s` bigint UNSIGNED NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"uint32": "`%s` int UNSIGNED NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"uint16": "`%s` smallint UNSIGNED NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"uint8":  "`%s` tinyint UNSIGNED NOT NULL DEFAULT '0' COMMENT '%s',\n",

	"float64": "`%s` float NOT NULL DEFAULT '0' COMMENT '%s',\n",
	"float32": "`%s` float NOT NULL DEFAULT '0' COMMENT '%s',\n",

	"string": "`%s` varchar(255) NOT NULL DEFAULT '' COMMENT '%s',\n",

	"time.Time": "`%s` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '%s',\n",

	"*create":  "CREATE TABLE `%s` (\n",
	"*end":     ")ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;\n",
	"*PRIMARY": "PRIMARY KEY (`%s`)\n",
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
func struct2Table(srcFile string, dsn *DsnConf) error {

	raw, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return err
	}
	filedMap, err := analysisSrc(BytesToString(raw))
	if err != nil {
		return err
	}
	sqlStrList := genTableSql(filedMap)
	for _, v := range sqlStrList {
		err = exeSql(v, dsn)
		if err != nil {
			return err
		}
	}
	return nil
}

type exeModel struct {
	DataBase string
	SqlSrc   string
}

func genTableSql(src map[string]Model) []exeModel {
	sqlStrList := make([]exeModel, 0)
	for _, v := range src {
		sqlStr := fmt.Sprintf(typeForGoToMysql["*create"], v.TableName)
		for _, f := range v.Fileds {
			sqlStr += fmt.Sprintf(typeForGoToMysql[f.Type], f.Tag, f.Annotation)
		}
		sqlStr += fmt.Sprintf(typeForGoToMysql["*PRIMARY"], "id")
		sqlStr += typeForGoToMysql["*end"]
		sqlStrList = append(sqlStrList, exeModel{
			DataBase: v.DataBase,
			SqlSrc:   sqlStr,
		})
	}
	return sqlStrList
}

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

func exportAnnotation(s string) string {
	idx := strings.Index(s, Annotation)
	if idx < 0 {
		return ""
	}
	if idx+2 < len(s) {
		s = s[idx+2:]
	} else {
		s = ""
	}
	return s
}
func analysisSrc(src string) (map[string]Model, error) {
	models := make(map[string]Model)
	r := regexp.MustCompile(`( )+|(\n)+`)
	src = r.ReplaceAllString(src, "$1$2")
	list := strings.Split(src, "\n")
	idx := 0
	strF := func(s string) string {
		s = strings.ReplaceAll(s, "\t", "")
		s = strings.TrimSpace(s)
		return s
	}
	for idx < len(list) {
		line := list[idx]
		if strings.Contains(line, "type") && strings.Contains(line, "struct") {
			tmp := strings.Split(line, " ")
			if len(tmp) < 2 {
				continue
			}
			structName := tmp[1]
			sM, ok := models[structName]
			if !ok {
				sM = Model{
					TableName: "",
					Struct:    structName,
					Fileds:    make([]structFiled, 0),
				}
			}
			for i := idx + 1; i < len(list); i++ {
				if list[i] == "}" {
					line = list[i]
					idx = i
					break
				}
				list[i] = strF(list[i])
				fileds := strings.Split(list[i], " ")
				if len(fileds) < 2 {
					continue
				}

				name := fileds[0]
				t := fileds[1]
				tagIndex := strings.Index(list[i], Tag)
				tag := ""
				if tagIndex > 0 {
					tagIndex += len(Tag) + 1
					str := list[i][tagIndex:]
					for _, item := range str {
						if item == '"' {
							break
						}
						tag += string(item)
					}
				}
				sM.Fileds = append(sM.Fileds, structFiled{
					Name:       name,
					Type:       t,
					Tag:        tag,
					Annotation: exportAnnotation(list[i]),
				})
			}
			models[structName] = sM
		}
		if strings.Contains(line, "func") && strings.Contains(line, "TableName") {
			line = strings.ReplaceAll(line, "\t", "")
			line = strings.TrimSpace(line)
			tmpList := strings.Split(line, "(")
			modelName := ""
			if len(tmpList) == 3 {
				for j, v := range tmpList {
					if j == 1 {
						tmpList = strings.Split(v, " ")
						if len(tmpList) > 1 {
							modelName = strings.ReplaceAll(tmpList[1], "*", "")
							modelName = strings.ReplaceAll(modelName, ")", "")
						}
						break
					}

				}
			}
			if modelName != "" {
				tName := ""
				for j := idx + 1; j < len(list); {
					list[j] = strings.ReplaceAll(list[j], "\t", "")
					list[j] = strings.TrimSpace(list[j])
					list[j] = strings.ReplaceAll(list[j], `"`, "")
					tName = strings.Split(list[j], " ")[1]
					idx = j
					break
				}
				if m, ok := models[modelName]; !ok {
					models[modelName] = Model{
						TableName: tName,
						Struct:    modelName,
						Fileds:    make([]structFiled, 0),
					}
				} else {
					m.TableName = tName
					models[modelName] = m
				}
			}
		}
		idx++
	}
	for k, v := range models {
		if v.TableName == "" {
			delete(models, k)
			continue

			//return models, errors.New("check src file should provide TableName function")
		}
		ts := strings.Split(v.TableName, ".")
		if len(ts) == 2 {
			v.DataBase = ts[0]
			v.TableName = ts[1]
		}
		for i := 0; i < len(v.Fileds); i++ {
			if v.Fileds[i].Tag == "" {
				v.Fileds[i].Tag = humpToFiledRule(v.Fileds[i].Name)
			}
		}
		models[k] = v
	}

	return models, nil
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
