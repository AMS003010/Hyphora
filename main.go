package main

import (
	"fmt"
	"log"

	"github.com/AMS003010/Hyphora/internal/bitcask"
)

func main() {
	// Open or create the Bitcask store in ./data
	bc, err := bitcask.Open("./data")
	if err != nil {
		log.Fatalf("failed to open bitcask: %v", err)
	}
	defer bc.Close()

	// Put some keys
	if err := bc.Put("foo2", []byte("bar")); err != nil {
		log.Fatal("put failed:", err)
	}
	if err := bc.Put("hello2", []byte("world")); err != nil {
		log.Fatal("put failed:", err)
	}
	if err := bc.Put("waasup2", []byte("Correct")); err != nil {
		log.Fatal("put failed:", err)
	}
	// if err := bc.Put("hahahaha", []byte("GOOOOOOOD")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }
	// if err := bc.Put("waasup2", []byte("kkk")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }
	// if err := bc.Put("waasup3", []byte("kkk")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }
	// if err := bc.Put("waasup4", []byte("kkk")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }
	// if err := bc.Put("waasup6", []byte("kkk")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }
	// if err := bc.Put("waasup7", []byte("kkk")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }
	// if err := bc.Put("waasup8", []byte("kkk")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }

	// Get them back
	val, err := bc.Get("waasup2")
	if err != nil {
		log.Println("get failed:", err)
	} else {
		fmt.Println("waasup2 =", string(val))
	}

	// val, err = bc.Get("hello")
	// if err != nil {
	// 	log.Println("get failed:", err)
	// } else {
	// 	fmt.Println("hello =", string(val))
	// }

	// if err := bc.Put("hello", []byte("juice")); err != nil {
	// 	log.Fatal("put failed:", err)
	// }

	// val, err = bc.Get("hello")
	// if err != nil {
	// 	log.Println("get failed:", err)
	// } else {
	// 	fmt.Println("hello =", string(val))
	// }

	// // Delete a key
	// if err := bc.Delete("foo"); err != nil {
	// 	log.Fatal("delete failed:", err)
	// }

	// // Try to get deleted key
	// _, err = bc.Get("foo")
	// fmt.Println("after delete foo:", err)

	// List keys
	fmt.Println("keys snapshot:", bc.Keys())
	fmt.Println("")
	fmt.Println("Bitcask : ", bc)
}
