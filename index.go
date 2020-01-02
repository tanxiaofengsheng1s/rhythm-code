package main

import (
	"fmt"
	"net/http"

	//  "encoding/json"
	"database/sql"
	//  "github.com/gin-gonic/gin"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	SIGN_NAME_SCERET = "aweQurt178BNI"
)

const IP_URL = "http://172.16.1.71:8000/"

//用户结构
type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	School   string `json:"school"`
}

//card内容
type Card struct {
	// Id          string  `json:"id"`
	Title   string `json:"title"`
	Image   string `json:"image"`
	Video   string `json:"video"`
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

//留言结构
type Comment struct {
	Card_id  string `json:"card_id"`
	User_id  string `json:"user_id"`
	Comments string `json:"comments"`
}

//登陆功能
func denglu(c echo.Context) error {

	cookie, err := c.Cookie("denglu")
	fmt.Println(cookie)
	if err != nil {
		user := new(User)
		if err = c.Bind(user); err != nil {
			return c.JSON(http.StatusOK, "数据错误")
		}
		if user.Name != "" && user.Password != "" {
			db, err := connect(c) //连接数据库
			if err != nil {
				return err
			}
			defer db.Close() //关闭数据库
			resp := map[string]string{"message": "登录失败！"}
			fmt.Println(user)
			result := db.QueryRow("select password,id,userimages from info where username=?", user.Name) //单行查询
			var password, id, userimage string
			result.Scan(&password, &id, &userimage)
			fmt.Println(result)
			if user.Password == password {
				tokenString, err := createJwt(id)
				if err != nil {
					fmt.Println(err.Error())
					return c.JSON(http.StatusOK, "token生成失败")
				}
				cookie := new(http.Cookie)
				cookie.Name = "denglu"
				cookie.Value = tokenString
				cookie.Expires = time.Now().Add(24 * time.Hour)
				c.SetCookie(cookie)
				resp = map[string]string{"message": "登录成功！", "name": user.Name, "userimage": userimage, "user_id": id, "token": tokenString}
			}

			return c.JSON(http.StatusOK, resp)
		} else {
			return c.JSON(http.StatusOK, "数据为空")
		}
	} else {
		fmt.Println(cookie)
		claims := parseJwt(cookie.Value)
		user_id := claims["user_id"]
		if user_id == "expired" {
			return delCookie(c)
		}
		db, err := connect(c) //连接数据库
		if err != nil {
			return err
		}
		defer db.Close()                                                                                                //关闭数据库
		result := db.QueryRow("select username,userimages,school,likes,collects,follows from info where id=?", user_id) //单行查询
		var name, userimage, school, likes, collects, follows string
		result.Scan(&name, &userimage, &school, &likes, &collects, &follows)
		resp := map[string]string{"message": "登录成功！", "user_id": user_id.(string), "name": name, "userimages": userimage, "school": school, "likes": likes, "collects": collects, "follows": follows}
		return c.JSON(http.StatusOK, resp)
	}

}

//删除cookie
func delCookie(c echo.Context) (err error) {
	cookie := new(http.Cookie)
	cookie.Name = "denglu"
	cookie.MaxAge = -1
	c.SetCookie(cookie)
	return c.JSON(http.StatusOK, "token已过期")
}

//获取信息接口
func showEverthing(c echo.Context) error {
	cardList := c.QueryParam("cardlist")
	cardID := c.QueryParam("card")
	userID := c.QueryParam("people")
	resp := map[string]string{"message": "获取信息失败！"}
	//获取用户信息
	if userID != "" {
		db, err := connect(c) //连接数据库
		if err != nil {
			return err
		}
		result := db.QueryRow("select username,userimages,collects,follows from info where id=?", userID) //单行查询用户基本信息
		var name, userimage, collects, follows string
		result.Scan(&name, &userimage, &collects, &follows)
		userimage = IP_URL + userimage
		collects = strings.TrimRight(collects, "|")
		collectsList := strings.Split(collects, "|")
		follows = strings.TrimRight(follows, "|")
		followsList := strings.Split(follows, "|")
		// fmt.Println(result)
		if name != "" {
			result, err := db.Query("select id,title,image,message,video,likes from card_list where user_id=?", userID)
			if err != nil {
				return err
			}
			defer db.Close() //关闭数据库
			i := 0
			cards := map[int]interface{}{}
			for result.Next() { //循环显示所有的数据

				var id, title, image, message, video, likes string
				result.Scan(&id, &title, &image, &message, &video, &likes)
				image = IP_URL + image
				card := map[string]interface{}{"id": id, "title": title, "message": message, "image": image, "video": video, "likes": likes}
				cards[i] = card
				i++

			}
			resp := map[string]interface{}{"name": name, "userimage": userimage, "cardsList": cards, "collects": collectsList, "follows": followsList}
			return c.JSON(http.StatusOK, resp)
		}
	}
	if cardList != "" {
		kind := c.QueryParam("kind")
		last := 0
		if cardList != "0" {
			cardList, _ := strconv.Atoi(cardList)
			last = cardList*5 + 1
		}
		db, err := connect(c) //连接数据库
		if err != nil {
			return err
		}
		result, err := db.Query("select id from card_list limit ?,?", last, 5)
		if kind == "article" {
			result, err = db.Query("select id from card_list where video='' limit ?,?", last, 5)
		} else if kind == "video" {
			result, err = db.Query("select id from card_list where video<>'' limit ?,?", last, 5)
		} else if kind == "hot" {
			result, err = db.Query("select id from card_list order by likes desc limit ?,?", last, 5)
		}
		if err != nil {
			return err
		}
		defer db.Close() //关闭数据库
		// fmt.Println(result)
		// columns, _ := result.Columns()
		// columnLength := len(columns)
		cards := map[int]interface{}{}
		i := 0
		for result.Next() { //循环显示所有的数据

			var id string
			result.Scan(&id)
			card := map[string]interface{}{"id": id}
			cards[i] = card
			i++
		}
		// card:=map[string]interface{}{"id":id,"title":title,"message":message}
		// cards:=map[int]interface{}{1:card,2:card}
		resp := map[string]interface{}{"cards": cards}
		return c.JSON(http.StatusOK, resp)

	}
	if cardID != "" {
		db, err := connect(c) //连接数据库
		if err != nil {
			return err
		}

		result := db.QueryRow("select id,title,image,message,video,user_id,likes,kind from card_list where id=? ", cardID)
		defer db.Close() //关闭数据库
		var id, title, image, message, video, user_id, likes, kind string
		result.Scan(&id, &title, &image, &message, &video, &user_id, &likes, &kind)
		likebool := false
		collectbool := false
		cookie, err := c.Cookie("denglu")
		if err != nil {
		} else {
			claims := parseJwt(cookie.Value)
			userID := claims["user_id"]
			result = db.QueryRow("select likes,collects from info where id=? ", userID)
			var userLikes, userCollects string
			result.Scan(&userLikes, &userCollects)
			likebool = strings.Contains(userLikes, cardID)
			collectbool = strings.Contains(userCollects, cardID)
		}
		if image != "" {
			image = IP_URL + image
		}
		if video != "" {
			video = IP_URL + video
		}
		res, err := db.Query("select content,author_id,datetime from comments where id=?", cardID)
		commits := map[int]interface{}{}
		i := 0
		for res.Next() { //循环显示所有的数据

			var comments, author_id, datetime string
			res.Scan(&comments, &author_id, &datetime)
			result = db.QueryRow("select username,userimages from info where id=? ", author_id)
			var username, userimages string
			result.Scan(&username, &userimages)
			comment := map[string]string{"author": username, "avatar": IP_URL + userimages, "content": comments, "datetime": datetime}
			commits[i] = comment
			i++
		}
		cards := map[string]interface{}{"id": id, "title": title, "message": message, "video": video, "image": image, "user_id": user_id, "likes": likes, "kind": kind}
		resp := map[string]interface{}{"cards": cards, "likes": likebool, "collects": collectbool, "comments": commits}
		return c.JSON(http.StatusOK, resp)
	}
	return c.JSON(http.StatusOK, resp)
}

//连接数据库
func connect(c echo.Context) (*sql.DB, error) {
	db, _ := sql.Open("mysql", "root:root@(127.0.0.1:3306)/data1") // 设置连接数据库的参数

	err := db.Ping() //连接数据库
	if err != nil {
		fmt.Println("数据库连接失败")
		return db, c.JSON(http.StatusOK, "数据库连接失败")
	}
	return db, err

}

// 获取文章列表
func artlist(c echo.Context) error {
	db, err := connect(c)
	result, err := db.Query("select  * from artlist ")
	cards := map[int]interface{}{}
	i := 0
	for result.Next() { //循环显示所有的数据

		var id, title, message, image, video, user_id, likes, name, comments, via string
		result.Scan(&id, &title, &message, &image, &video, &user_id, &likes, &name, &comments, &via)
		// image = IP_URL+image
		card := map[string]interface{}{"id": id, "title": title, "image": image, "message": message, "video": video, "user_id": user_id, "likes": likes, "name": name, "comments": comments, "via": via}
		cards[i] = card
		i++

	}
	resp := map[string]interface{}{"cards": cards}
	if err != nil {
		log.Fatalln(err.Error())
	}
	Response := resp

	return c.JSON(http.StatusOK, Response)
}

// 获取书籍列表
func booklist(c echo.Context) error {
	db, err := connect(c)
	result, err := db.Query("select  * from book ")
	books := map[int]interface{}{}
	i := 0
	for result.Next() { //循环显示所有的数据

		var id, imageinfo, desc, author, price, likes string
		result.Scan(&id, &imageinfo, &desc, &author, &price, &likes)
		// image = IP_URL+image
		book := map[string]interface{}{"id": id, "imageinfo": imageinfo, "desc": desc, "author": author, "price": price, "likes": likes}
		books[i] = book
		i++

	}
	resp := map[string]interface{}{"books": books}
	if err != nil {
		log.Fatalln(err.Error())
	}
	Response := resp

	return c.JSON(http.StatusOK, Response)
}

func main() {
	e := echo.New()
	//跨域中间件
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://172.16.1.71:3000"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderXRequestedWith},
		AllowCredentials: true,
	}))

	e.POST("/denglu", denglu) //登陆路由

	e.GET("/show", showEverthing)    //获取信息接口
	e.GET("/artlist", artlist)       //获取文章列表
	e.GET("/booklist", booklist)     //获取书籍
	e.Static("/avator", "./avator")  //用户头像资源
	e.Static("/images", "./images")  //用户图片资源
	e.Static("/media", "./media")    //用户视频资源
	e.Logger.Fatal(e.Start(":8000")) //端口号8000

}

//创建 token
func createJwt(id string) (string, error) {
	//	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	//		"foo": "bar",
	//		"nbf": time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),

	//	})
	token := jwt.New(jwt.SigningMethodHS256)
	claims := make(jwt.MapClaims)
	claims["user_id"] = id
	claims["exp"] = time.Now().Add(6 * time.Hour * time.Duration(1)).Unix()
	claims["iat"] = time.Now().Unix()
	token.Claims = claims

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(SIGN_NAME_SCERET))
	return tokenString, err
}

//解析 token
func parseJwt(tokenString string) jwt.MapClaims {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(SIGN_NAME_SCERET), nil
	})

	var claims jwt.MapClaims
	var ok bool

	if claims, ok = token.Claims.(jwt.MapClaims); ok && token.Valid {
		// fmt.Println(claims["user_id"], claims["nbf"])
		return claims
	} else {
		fmt.Println(err)
		claims["user_id"] = "expired"
	}
	return claims
}
