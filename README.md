# golang struct与mysql table 的转换工具 
- 添加一个链接配置
  ```bash
  # ./godbtool add alias ip port user password
  ./godbtool add local 127.0.0.1 3306 root 12345678
  ```

- 删除一个配置

  ```bash
   # ./godbtool del alias
   ./godbtool del local
  ```

- 表结构转换数据结构

  ```bash
  # ./godbtool tostruct alias database.table_name file_path
  ./godbtool tostruct local demo.goadmin_operation_log ./model.go
  ```

- 数据结构转化表结构，根据配置自动创建表

  ```bash
  # ./godbtool totable alias target_src.go
  ./godbtool totable local model.go
  ```

  配置文件路径

  ```bash
  ~/.godbtool.json
  ```

 暂停维护	 
