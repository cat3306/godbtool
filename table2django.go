package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var typeForMysqlToDjango = map[string]string{
	"int": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"integer": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"tinyint": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"smallint": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"mediumint": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"bigint": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"int unsigned": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"integer unsigned": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"tinyint unsigned": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"smallint unsigned": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"mediumint unsigned": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"bigint unsigned": `    %s = models.IntegerField(verbose_name="%s",default=%s)
`,
	"varchar": `    %s = models.CharField(verbose_name="%s", max_length=%s,default="%s")
`,
	"char": `    %s = models.CharField(verbose_name="%s", max_length=%s,,default="%s")
`,
	"tinytext": `    %s = models.TextField(verbose_name="%s", default=None)
`,
	"mediumtext": `    %s = models.TextField(verbose_name="%s", default=None)
`,
	"text": `    %s = models.TextField(verbose_name="%s", default=None)
`,
	"longtext": `    %s = models.TextField(verbose_name="%s", default=None)
`,
	"date": `    %s = models.DateField()
`, // time.Time or string
	"datetime": `    %s = models.DateTimeField()
`, // time.Time or string
	"timestamp": `    %s = models.DateTimeField()
`, // time.Time or string
	"time": `    %s = models.DateTimeField()
`, // time.Time or string
	"float": `    %s = models.FloatField(verbose_name="%s", default=%s)
`,
	"double": `    %s = models.FloatField(verbose_name="%s", default=%s)
`,
	"decimal": `    %s = models.FloatField(verbose_name="%s", default=%s)
`,
	"binary": `    %s = models.BinaryField()
`,
	"first": `class %s(models.Model):
`,
	"last": `    class Meta:
	managed = False
	db_table = '%s'
	verbose_name_plural = verbose_name = ''

	def __str__(self):
		return str(self.pk)
`,
}

func NewTable2Django() *Table2Django {
	return &Table2Django{}
}

type Table2Django struct {
	dsn      string
	dataBase string
	db       *sql.DB
	err      error
	table    string
}

func (t *Table2Django) Dsn(c *DsnConf) *Table2Django {
	t.dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", c.User, c.Pwd, c.Ip, c.Port, c.DataBase)
	t.dataBase = c.DataBase
	return t
}
func (t *Table2Django) Table(table string) *Table2Django {
	t.table = table
	return t
}
func (t *Table2Django) dialMysql() {
	if t.db == nil {
		if t.dsn == "" {
			t.err = errors.New("dsn数据库配置缺失")
			return
		}
		t.db, t.err = sql.Open("mysql", t.dsn)
	}
	return
}
func (t *Table2Django) Run() error {
	t.dialMysql()
	if t.err != nil {
		return t.err
	}
	return t.getColumns()

}
func (t *Table2Django) getColumns() error {
	var sqlStr = `SELECT COLUMN_NAME,DATA_TYPE,IS_NULLABLE,TABLE_NAME,COLUMN_COMMENT,COLUMN_DEFAULT,CHARACTER_MAXIMUM_LENGTH as column_length
		FROM information_schema.COLUMNS 
		WHERE table_schema = DATABASE()`
	// 是否指定了具体的table
	if t.table != "" {
		sqlStr += fmt.Sprintf(" AND TABLE_NAME = '%s'", t.table)
	}
	// sql排序
	sqlStr += " order by TABLE_NAME asc, ORDINAL_POSITION asc"
	rows, err := t.db.Query(sqlStr)
	if err != nil {
		return err
	}
	defer rows.Close()
	tableColumns := make(map[string][]column)
	for rows.Next() {
		col := column{}
		err = rows.Scan(&col.ColumnName, &col.Type, &col.Nullable, &col.TableName, &col.ColumnComment, &col.ColumnDefault, &col.ColumnLength)
		if _, ok := tableColumns[col.TableName]; !ok {
			tableColumns[col.TableName] = []column{}
		}
		tableColumns[col.TableName] = append(tableColumns[col.TableName], col)
	}
	var targets []string
	for tName, col := range tableColumns {
		s := fmt.Sprintf(typeForMysqlToDjango["first"], tName)
		for _, v := range col {
			tmp := typeForMysqlToDjango[v.Type]
			if v.ColumnDefault == "" {
				if strings.Contains(tmp, "IntegerField") || strings.Contains(tmp, "FloatField") {
					v.ColumnDefault = "0"
				}
			}
			if v.ColumnLength == "" {
				v.ColumnLength = "255"
			}

			cnt := strings.Count(tmp, "%")
			if cnt == 1 {
				s += fmt.Sprintf(tmp, v.ColumnName)
			}
			if cnt == 2 {
				s += fmt.Sprintf(tmp, v.ColumnName, v.ColumnComment)
			}
			if cnt == 3 {
				if strings.Contains(tmp, "IntegerField") {
					if v.ColumnDefault == "" {
						v.ColumnDefault = "0"
					}
					s += fmt.Sprintf(tmp, v.ColumnName, v.ColumnComment, v.ColumnDefault)
				}
			}
			if cnt == 4 {
				s += fmt.Sprintf(tmp, v.ColumnName, v.ColumnComment, v.ColumnLength, v.ColumnDefault)
			}
		}
		s += fmt.Sprintf(typeForMysqlToDjango["last"], tName)
		targets = append(targets, s)
	}
	fmt.Println(targets[0])
	return nil
}
func (t *Table2Django) genDjangoCode() {

}
