package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func getHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Success",
	})
}

func main() {

	fmt.Println("Listening and Serving on Port 8000")

	router := http.NewServeMux()
	router.HandleFunc("GET /health", getHealth)

	err := http.ListenAndServe(":8000", router)

	if err != nil {
		fmt.Println("Error Starting the Server")
	}
}
