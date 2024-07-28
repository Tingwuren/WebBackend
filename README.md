# Web后端开发技术大作业说明
## 环境说明
1. 开发语言：Go（版本1.22）
2. 开发工具：GoLand集成开发环境
3. 数据库工具：
	1. SQLite3：数据持久化
	2. Redis：数据缓存
4. 部署工具：
	1. Docker Compose：部署多个服务
	2. Docker：部署单个服务
	3. Nginx：服务负载均衡
5. 测试工具：
	1. Apifox：包含Postman接口文档化和Jmeter自动化测试
## 部署说明
1. 本地单机部署：
	1. 使用GoLand打开项目文件夹backend
	2. 打开设置，选择Go->GOROOT为1.22版本
	3. 本地启动Redis，端口号为6379
	4. 运行go build backend
2. Docker多服务部署：
	1. 使用GoLand打开项目文件夹backend
	2. 修改Redis数据库连接信息，由于使用Docker部署Redis，实现容器间的相互通信，修改main.go第53行为`Addr: "redis:6379",`
	3. 修改SQLite数据库连接信息，修改main.go第78行为`db, _ = gorm.Open(sqlite.Open("/app/data/gorm.db"), &gorm.Config{})`，代表多个服务共享同一个数据库。
	4. 在项目文件夹下打开终端，输入命令`docker-compose up --build`，启动一个Nginx，一个Redis，四个后端服务
## 测试说明
1. 本地单机部署说明：
	1. 使用Apifox导入接口文档backend.apifox.json，导入数据格式为Apifox，导入12个接口，2个环境，2个测试场景
	2. 选择环境为backend course
	3. 使用接口管理可以测试各个接口的功能
	4. 使用自动化测试可以测试Redis缓存能力
2. Docker多服务部署：
	1. 使用Apifox导入接口文档backend.apifox.json，如果导入过就不再导入
	2. 选择环境为docker-backend
	3. 使用接口管理可以测试各个接口的功能
	4. 使用自动化测试可以测试Redis缓存能力和Nginx负载均衡能力