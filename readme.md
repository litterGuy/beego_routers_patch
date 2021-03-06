# beego 注解路由生成补丁

说明：
    以下几个参数可放置到配置文件
    
```
//生成路由文件的前缀名称
ROUTER_PREFIX         = "commentsRouter_"
//项目url的统一前缀
PROJECT_ROUTER_PREFIX = "/v1"
//源码相对于根目录的存放位置
PROJECT_SOURCE_CATALOG = "src"
//项目固定的名称（因想直接修改项目文件夹名称，不去修改go.mod 而项目能正常运行）
PEOJECT_NAME = "baseadmin"  
```  
1、所有的 controller 需要放置到 "controllers"包下，可以在该包下包含不同的文件夹 
```go
    ─controllers
    │  ├─admin
    │  └─system
    ├─routers
    │  ├─router.go
```    

2、"routers" 目录和 "controllers" 包同级

3、需要在 "main.go" 主文件引入 routers 目录做初始化，如
```go
_ "baseadmin/src/routers"
```

4、需要在 controller 的 struct 结构体添加注解 //@action，（也可不添加直接用@router）如
```go
//@action /account
type LoginController struct {
    beego.Controller
}

//@action 
type LoginController struct {
    beego.Controller
}
```
也可直接直接添加注解，没有路由前缀

5、在对应的函数增加注解//@router
```go
//@router	/login	[post]
func (l *LoginController) Login() {
}

//@router	/login	[get,post]
func (l *LoginController) Login() {
}

//@router	/login/:key	[get,post]
func (l *LoginController) Login() {
}

```

6、需要注意以下几点限制
    
    1. go文件中 "package ******" 不能换行，否则出错
    2. 尽量保证项目格式化，混乱的文件格式或许会导致扫描异常
    3. 新增的路由在刚启动后生成文件，但是无法生效、需要重新启动一次项目