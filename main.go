package main

func main() {
	go PullMsgsSync("yarb-313112", "yarb-telegram")
	r := setupRouter()
	r.Run(":8070")
}
