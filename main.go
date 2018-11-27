package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
        "encoding/json"
        "strconv"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type User struct {
        Id    int
        Name  string
        Address string
        Email string
        Role string
        Password string 
} 


var db *sql.DB

func main() {
	var err error
        connStr := "user=shop_user password=shoppass dbname=shop sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	fatal(err)

	router := httprouter.New()
	router.GET("/api/v1/users", getUsers)
        router.POST("/api/v1/user", addUser)
        router.DELETE("/api/v1/user/:id", deleteUser)

	http.ListenAndServe(":8080", router)
}




func read () ([]User, error) {
        rows, err := db.Query("SELECT u.id,u.email,u.address,r.role FROM users u JOIN roles r ON u.role=r.id")
        if err != nil {
           panic(err)
        }
        defer rows.Close()
        users := []User{}
        var u User
        for rows.Next(){
            err := rows.Scan(&u.Id, &u.Email,&u.Address,&u.Role)
            if err != nil{
              fmt.Println(err)
            }
            users = append(users, u)
        }

        return users, nil
}



                
func getUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
        usrs, err := read() 
        if err != nil {
          w.WriteHeader(500)
        }  
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(usrs)
}

func insert(email, password string) (sql.Result, error) {
        return db.Exec("INSERT INTO users (email, password) VALUES ($1, $2)",
                email, password)
}


func addUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
        var user User
        decoder := json.NewDecoder(r.Body)
        err := decoder.Decode(&user)
        fatal(err)
        if _, err := insert(user.Email, user.Password); err != nil {
                w.WriteHeader(500)
                return
        }
        w.WriteHeader(201)
}


func getID(w http.ResponseWriter, ps httprouter.Params) (int, bool) {
        id, err := strconv.Atoi(ps.ByName("id"))
        if err != nil {
                w.WriteHeader(400)
                return 0, false
        }
        return id, true
}


func remove(id int) (sql.Result, error){
      return db.Exec("DELETE FROM users WHERE id=$1", id)
}


func deleteUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
        id, ok := getID(w, ps)
        if !ok {
                return
        }
        if _, err := remove(id); err != nil {
                w.WriteHeader(500)
        }
        w.WriteHeader(204)
}

