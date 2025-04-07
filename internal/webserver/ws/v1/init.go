package v1

func init() {
	go broadcastContainerList()
	go broadcastContainerStats()
}
