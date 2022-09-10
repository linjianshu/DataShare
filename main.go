package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

func main() {
	//展示主界面
	http.HandleFunc("/showMainIndex", showMainIndex)

	//登录或者注册
	http.HandleFunc("/loginOrRegister", loginOrRegister)

	//展示登录界面
	http.HandleFunc("/showLoginIndex", showLoginIndex)

	//上传文件界面
	http.HandleFunc("/IndexForUpload", showIndexForUpload)

	//上传文件
	http.HandleFunc("/uploadFile", uploadFile)

	http.HandleFunc("/list", fileServer)

	http.ListenAndServe(":8888", nil)
}

// showMainIndex 展示主页面
func showMainIndex(w http.ResponseWriter, r *http.Request) {
	//权限验证
	if !AuthorityCheck(w, r) {
		http.Redirect(w, r, "/showLoginIndex", http.StatusFound)
		return
	}

	t := template.Must(template.New("mainIndex.html").ParseFiles("mainIndex.html"))
	t.Execute(w, nil)
}

func loginOrRegister(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	userpwd := r.FormValue("userpwd")
	if username == "" || userpwd == "" {
		http.Redirect(w, r, "/showLoginIndex", http.StatusFound)
		//http.Error(w, "username is null or userpwd is null", http.StatusBadRequest)
		return
	}
	DBConnect()

	var correctPwd string
	rows, err := Db.Query("select userpwd from user where username = ? ", username)
	if err != nil {
		http.Redirect(w, r, "/showLoginIndex", http.StatusFound)
		//msg := fmt.Sprintf("get database userpwd failed, error : %v\n", err.Error())
		//http.Error(w, msg, http.StatusNotFound)
		return
	}

	//如果找不到一行的话 就进来
	if !rows.Next() {
		//这里写注册的逻辑
		exec, err := Db.Exec("insert into user values (? , ? , ? , ?,?,?)", username, userpwd, "", "", "", "")
		if err != nil {
			return
		}
		affected, err := exec.RowsAffected()
		if err != nil {
			return
		}
		if affected > 0 {
			setCookie(w, username, userpwd)
			http.Redirect(w, r, "/showMainIndex", http.StatusTemporaryRedirect)
			return
		} else {
			return
		}
	}
	rows.Scan(&correctPwd)
	if correctPwd == userpwd {
		setCookie(w, username, userpwd)
		http.Redirect(w, r, "/showMainIndex", http.StatusTemporaryRedirect)
		return
	}

	http.Error(w, "incorrect username or userpwd.\n", http.StatusNotFound)
	return
}

func setCookie(w http.ResponseWriter, username string, userpwd string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "username",
		Value:    username,
		MaxAge:   60,
		HttpOnly: false,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "userpwd",
		Value:    userpwd,
		MaxAge:   60,
		HttpOnly: false,
	})
}

func showLoginIndex(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("index2.html").ParseFiles("index2.html"))
	t.Execute(w, nil)
}

func showIndexForUpload(w http.ResponseWriter, r *http.Request) {
	//权限验证
	if !AuthorityCheck(w, r) {
		http.Redirect(w, r, "/showLoginIndex", http.StatusPermanentRedirect)
		return
	}

	t := template.Must(template.New("ShowIndexForUpload.html").ParseFiles("ShowIndexForUpload.html"))
	t.Execute(w, nil)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	//权限验证
	if !AuthorityCheck(w, r) {
		http.Redirect(w, r, "/showLoginIndex", http.StatusPermanentRedirect)
		return
	}

	r.ParseMultipartForm(1 << 32)
	r.ParseForm()
	_, header, err := r.FormFile("uploadify")
	if err != nil {
		w.Write([]byte("trying to get formFile error : "))
		w.Write([]byte(err.Error()))
		return
	}
	open, err := header.Open()
	if err != nil {
		w.Write([]byte("trying to open target file error : "))
		w.Write([]byte(err.Error()))
		return
	}
	defer open.Close()

	fileName := path.Join("D:\\MESFiles", header.Filename)
	newFile, err := os.Create(fileName)
	if err != nil {
		w.Write([]byte("trying to create file on remote server error : "))
		w.Write([]byte(err.Error()))
		return
	}
	_, err = io.Copy(newFile, open)
	if err != nil {
		w.Write([]byte("trying to create file on remote server error : "))
		w.Write([]byte(err.Error()))
		return
	}
	defer newFile.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success !!!"))
}

func fileServer(w http.ResponseWriter, r *http.Request) {
	//权限验证
	if !AuthorityCheck(w, r) {
		http.Redirect(w, r, "/showLoginIndex", http.StatusPermanentRedirect)
		return
	}

	r.URL.Path = "/"
	server := http.FileServer(http.Dir("D:\\MESFiles"))
	//http.Handle("/", http.fileServer(http.Dir("home/project/aligo/DataShare/dist")))
	server.ServeHTTP(w, r)
}
func AuthorityCheck(w http.ResponseWriter, r *http.Request) bool {
	DBConnect()

	username, err := r.Cookie("username")
	if err != nil {
		//msg := fmt.Sprintf("get cookie username failed, error : %v\n", err.Error())
		//http.Error(w, msg, http.StatusNotFound)
		return false
	}

	userpwd, err := r.Cookie("userpwd")
	if err != nil {
		msg := fmt.Sprintf("get cookie userpwd failed, error : %v\n", err.Error())
		http.Error(w, msg, http.StatusNotFound)
		return false
	}

	var correctPwd string
	err = Db.QueryRow("select userpwd from user where username = ? ", username.Value).Scan(&correctPwd)
	if err != nil {
		msg := fmt.Sprintf("get database userpwd failed, error : %v\n", err.Error())
		http.Error(w, msg, http.StatusNotFound)
		return false
	}

	if correctPwd == userpwd.Value {
		return true
	}

	http.Error(w, "incorrect username or userpwd.\n", http.StatusNotFound)
	return false

}

var Db *sql.DB

func DBConnect() {
	//go连接mysql实例
	//数据库源信息
	dsn := "root:123456@tcp(127.0.0.1:3306)/datashare?charset=utf8&parseTime=true"

	var err error
	//连接数据库
	Db, err = sql.Open("mysql", dsn) //不会校验用户名和密码是否正确
	if err != nil {
		log.Printf("dsn %s invalid , err : %v\n", dsn, Db)
		return
	}
	err = Db.Ping()
	if err != nil {
		log.Printf("open %s failed , err : %v\n", dsn, err)
		return
	}
	fmt.Println(Db, "连接数据库成功!")
}
