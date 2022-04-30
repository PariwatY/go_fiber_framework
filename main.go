package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	jwtware "github.com/gofiber/jwt/v2"
)

var db *sqlx.DB

const jwtSecret = "jwtSecret"

func main() {
	var err error
	db, err = sqlx.Open("mysql", "root:1234@tcp(localhost:3306)/go_pek")
	if err != nil {
		panic(err)
	}
	app := fiber.New()

	app.Use("/hello", jwtware.New(jwtware.Config{
		SigningMethod: "HS256",
		SigningKey:    []byte(jwtSecret),
		SuccessHandler: func(c *fiber.Ctx) error {
			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return fiber.ErrUnauthorized
		},
	}))

	app.Post("/signup", Signup)
	app.Post("/login", Login)
	app.Get("/hello", Hello)

	app.Listen(":8081")
}

func Signup(c *fiber.Ctx) error {
	request := SignupRequest{}
	err := c.BodyParser(&request)
	if err != nil {
		return err
	}

	if request.Username == "" || request.Password == "" {
		return fiber.ErrUnprocessableEntity
	}

	password, err := bcrypt.GenerateFromPassword([]byte(request.Password), 10)
	if err != nil {
		return fiber.ErrUnprocessableEntity
	}

	query := "insert into user (username,password) values(?,?)"
	result, err := db.Exec(query, request.Username, string(password))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	user := User{
		Id:       int(id),
		Username: request.Username,
		Password: string(password),
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}
func Login(c *fiber.Ctx) error {

	request := LoginRequest{}

	err := c.BodyParser(&request)
	if err != nil {
		return fiber.ErrUnprocessableEntity
	}

	user := User{}
	query := "select username,password from user where username = ?"

	err = db.Get(&user, query, request.Username)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Incorrect Username")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Incorrect Username")
	}

	claims := jwt.StandardClaims{
		Issuer:    strconv.Itoa(user.Id),
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"jwtToken": token,
	})
}
func Hello(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"Hello": "Hello Wolrd!!!!!!!!!!!!!!",
	})
}

type User struct {
	Id       int    `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Password string `db:"password" json:"password"`
}

type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Fiber() {
	app := fiber.New(fiber.Config{
		Prefork: false,
	})

	//Middleware
	app.Use("/hello", func(c *fiber.Ctx) error {
		c.Locals("name", "bond")
		// fmt.Println("Before")
		err := c.Next()
		// fmt.Println("After")
		return err
	})

	//Middleware RequestID
	app.Use(requestid.New())

	//Middleware Cors
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
		AllowHeaders:     "",
		AllowCredentials: false,
	}))

	//Middleware Loggers
	app.Use(logger.New(logger.Config{
		TimeZone: "Asia/Bangkok",
	}))

	//GET
	app.Get("/hello", func(c *fiber.Ctx) error {
		name := c.Locals("name")
		// fmt.Println("Hello")
		return c.SendString(fmt.Sprintf("name: %v", name))
	})

	//POST
	app.Post("/hello", func(c *fiber.Ctx) error {
		return c.SendString("post hello")
	})

	//Parameter
	app.Get("/hello/name/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		return c.SendString("name: " + name)
	})

	//Parameter Optional
	app.Get("/hello/:name/:surname", func(c *fiber.Ctx) error {
		name := c.Params("name")
		surname := c.Params("surname")
		return c.SendString("name: " + name + " surname: " + surname)
	})

	//Parameter Integer
	app.Get("/hello/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return fiber.ErrBadRequest
		}
		return c.SendString(fmt.Sprintf("id: %v", id))
	})

	//Query
	app.Get("/query", func(c *fiber.Ctx) error {
		name := c.Query("name")
		surname := c.Query("surname")
		return c.SendString("name: " + name + ", surname: " + surname)
	})

	//Query2
	app.Get("/query2", func(c *fiber.Ctx) error {
		person := Person{}
		c.QueryParser(&person)
		return c.JSON(person)
	})

	//Wildcard
	app.Get("/wildcards/*", func(c *fiber.Ctx) error {
		wildcard := c.Params("*")
		return c.SendString(wildcard)
	})

	//Static File
	app.Static("/", "./wwwroot")

	//NewError
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.ErrForbidden.Code)
	})

	//Group
	v1 := app.Group("/v1")
	v1.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello V1")
	})

	v2 := app.Group("/v2")
	v2.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello V2")
	})

	//Mount
	userApp := fiber.New()
	userApp.Get("/login", func(c *fiber.Ctx) error {
		return c.SendString("Login")
	})
	app.Mount("/user", userApp)

	//Server
	app.Server().MaxConnsPerIP = 1
	app.Get("/server", func(c *fiber.Ctx) error {
		time.Sleep(time.Second * 30)
		return c.SendString("Server")
	})

	app.Get("/env", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"BaseURL":     c.BaseURL(),
			"Hostname":    c.Hostname(),
			"IP":          c.IP(),
			"IPs":         c.IPs(),
			"OriginalURL": c.OriginalURL(),
			"Path":        c.Path(),
			"Protocol":    c.Protocol(),
			"Subdomains":  c.Subdomains(),
		})
	})

	//Body
	app.Post("/body", func(c *fiber.Ctx) error {
		fmt.Printf("IsJson: %v\n", c.Is("json"))

		person := Person{}
		err := c.BodyParser(&person)
		if err != nil {
			return err
		}

		fmt.Println(person)
		return nil
	})

	app.Post("/body2", func(c *fiber.Ctx) error {
		fmt.Printf("IsJson: %v\n", c.Is("json"))

		data := map[string]interface{}{}
		err := c.BodyParser(&data)
		if err != nil {
			return err
		}

		fmt.Println(data)
		return nil
	})

	app.Listen(":8081")
}

type Person struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
