package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
        "encoding/json"
        "github.com/satori/go.uuid"
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
        defer db.Close()
	
          fmt.Println("Starting server...")
     

        router := httprouter.New()
	router.GET("/api/v1/users", getUsers)
        router.POST("/api/v1/user", addUser)
        router.DELETE("/api/v1/user/:id", deleteUser)
        router.POST("/api/v1/user/get_token", getToken)

	http.ListenAndServe(":8080", router)
}


func getToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
                decoder := json.NewDecoder(r.Body)
                var g_user User

                err := decoder.Decode(&g_user)
                if err != nil {
                        panic(err)
                }
               
                id, err := check_user(g_user.Email, g_user.Password)
                if err != nil {
		  http.Error(w, "User Not Found", 403)
                } else {
	          query := fmt.Sprintf("select token from session where user_id = %d and ((added + interval '600s') > now())", id)
		  row := db.QueryRow(query)

		  var token string
		  err = row.Scan(&token)
                  if err != nil {
	            token := uuid.Must(uuid.NewV4())
		    fmt.Println("# GOT TOKEN: %s", token)
		    query := fmt.Sprintf("INSERT INTO session (user_id, token) VALUES (%d, '%s')", id, token)
		    _, err := db.Exec(query)
		    if err != nil {
		      http.Error(w, "Internal Error", 500)
		    } else {
		      fmt.Fprintf(w, "{\"token\":%s}", token)
		    }

		  } else {
	            fmt.Fprintf(w, "{\"token\":\"%s\"}", token)
		  }
	        }

}


func check_user (email, password string) (int, error) {
        row := db.QueryRow("select id from users where email=$1 and password=$2", email, password)
        var id int
        err := row.Scan(&id)
        if err != nil {
          return 0, err
        } else {
          return id, err
        }

}

func is_admin(token string) bool{
       row := db.QueryRow("select u.role from session s join users u on s.user_id=u.id where token = $1 and ((added + interval '600s') > now())", token)
       var role int
       err := row.Scan(&role)
       if err != nil {
        fmt.Println(err)
       }
       if role != 1 {
       return false
       }  else {
       return true
       }
}

func same_user(token string, id int) bool{
       row := db.QueryRow("select user_id from session where token = $1 and ((added + interval '600s') > now())", token)
       var token_id int
       err := row.Scan(&token_id)
       if err != nil {
        fmt.Println(err)
       }
       if token_id != id {
       return false
       }  else {
       return true
       }
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
        token := r.Header.Get ("X-User-Token")
        if is_admin(token){  
        usrs, err := read() 
        if err != nil {
          w.WriteHeader(500)
        }  
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(usrs)
        } else {
         http.Error(w, "Access denied", 403)
        }   
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
        _, err = check_user (user.Email, user.Password)
        if err != nil {
           if _, err := insert(user.Email, user.Password); err != nil {
                w.WriteHeader(500)
                return
           }
           w.WriteHeader(201)
        } else {
          http.Error(w, "User Already Exist", 500)
        }
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
       
        token := r.Header.Get ("X-User-Token")
        if (is_admin(token) || same_user(token, id)) {
        if _, err := remove(id); err != nil {
                w.WriteHeader(500)
        }
        w.WriteHeader(204)
        } else {
           http.Error(w, "Access denied", 403)
        }
}


