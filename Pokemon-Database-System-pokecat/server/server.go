package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Constants
const (
	GridSize             = 10
	PokemonPerWave       = 10
	PokemonSpawnInterval = time.Minute
)

// Types
type Coord struct {
	X int
	Y int
}

type Player struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	PokemonList  []Pokemon `json:"pokemon_list"`
	CurrentCoord Coord
	mu           sync.Mutex
}

type Pokemon struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Type       []string `json:"type"`
	BaseExp    int      `json:"base_exp"`
	Speed      int      `json:"speed"`
	Attack     int      `json:"attack"`
	Defense    int      `json:"defense"`
	SpecialAtk int      `json:"special_atk"`
	SpecialDef int      `json:"special_def"`
	HP         int      `json:"hp"`
	EV         float64  `json:"ev"`
	CurrentExp int      `json:"current_exp"`
	Level      int      `json:"level"`
	SpawnTime  time.Time
	Coord      Coord
}

// Global Variables
var (
	players          []Player
	pokemons         []Pokemon
	playerIDCounter  = 0
	pokemonIDCounter = 0
	pokemonFile      = "pokedex.json"
	playerFile       = "player.json"
)

// Entry point of the server
func main() {
	rand.Seed(time.Now().UnixNano())
	loadGameData()

	go pokemonSpawner()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer listener.Close()

	fmt.Println("Server started. Listening on :8080")

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Enter 'exit' to stop the server.")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			if text == "exit" {
				fmt.Println("Exiting server...")
				os.Exit(0)
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			return
		}
		fmt.Println("Client connected.")

		go handleClient(conn)
	}
}

// Handle client connections
func handleClient(conn net.Conn) {
	defer conn.Close()

	conn.Write([]byte("Enter your name to register or login:\n"))
	name, err := readCommand(conn)
	if err != nil {
		fmt.Println("Error reading name:", err.Error())
		return
	}

	player := getPlayerByName(name)
	if player == nil {
		player = initializePlayer(name)
	}

	for {
		cmd, err := readCommand(conn)
		if err != nil {
			fmt.Println("Error reading command:", err.Error())
			return
		}

		switch cmd {
		case "move up":
			movePlayer(player, player.CurrentCoord.X, player.CurrentCoord.Y+1)
		case "move down":
			movePlayer(player, player.CurrentCoord.X, player.CurrentCoord.Y-1)
		case "move left":
			movePlayer(player, player.CurrentCoord.X-1, player.CurrentCoord.Y)
		case "move right":
			movePlayer(player, player.CurrentCoord.X+1, player.CurrentCoord.Y)
		case "capture":
			capturePokemon(player)
		case "show pokemons":
			showPokemons(conn, player)
		case "exit":
			fmt.Println("Player", player.Name, "exited.")
			return
		default:
			fmt.Println("Unknown command from player", player.Name)
		}

		sendPlayerStatus(conn, player)
	}
}

// Read command from client
func readCommand(conn net.Conn) (string, error) {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

// Send player status to client
func sendPlayerStatus(conn net.Conn, player *Player) {
	player.mu.Lock()
	defer player.mu.Unlock()

	status := fmt.Sprintf("Player: %s\nCoordinates: (%d, %d)\nPokemons: %d/%d\n",
		player.Name, player.CurrentCoord.X, player.CurrentCoord.Y, len(player.PokemonList), 200)
	conn.Write([]byte(status + "\n"))
}

// Show currently spawned Pokemons
func showPokemons(conn net.Conn, player *Player) {
	pokemonList := "Captured Pokemons:\n"
	for _, pokemon := range player.PokemonList {
		pokemonList += fmt.Sprintf("ID: %d, Name: %s, \n", pokemon.ID, pokemon.Name)
	}
	conn.Write([]byte(pokemonList + "\n"))
}

// Load initial game data from files
func loadGameData() {
	loadPlayers()
	loadPokemons()
}

// Load player data from file
func loadPlayers() {
	data, err := ioutil.ReadFile(playerFile)
	if err != nil {
		fmt.Println("Error reading player file:", err.Error())
		return
	}
	err = json.Unmarshal(data, &players)
	if err != nil {
		fmt.Println("Error unmarshalling player data:", err.Error())
		return
	}
	if len(players) > 0 {
		playerIDCounter = players[len(players)-1].ID
	}
	fmt.Println("Players loaded:", len(players))
}

// Load Pokemon data from file
func loadPokemons() {
	data, err := ioutil.ReadFile(pokemonFile)
	if err != nil {
		fmt.Println("Error reading pokemon file:", err.Error())
		return
	}
	err = json.Unmarshal(data, &pokemons)
	if err != nil {
		fmt.Println("Error unmarshalling pokemon data:", err.Error())
		return
	}
	fmt.Println("Pokemons loaded:", len(pokemons))
}

// Save player data to file
func savePlayers() {
	data, err := json.MarshalIndent(players, "", " ")
	if err != nil {
		fmt.Println("Error marshalling players data:", err.Error())
		return
	}
	err = ioutil.WriteFile(playerFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing player file:", err.Error())
		return
	}
	fmt.Println("Players saved:", len(players))
}

// Save Pokemon data to file
func savePokemons() {
	data, err := json.MarshalIndent(pokemons, "", " ")
	if err != nil {
		fmt.Println("Error marshalling pokemons data:", err.Error())
		return
	}
	err = ioutil.WriteFile(pokemonFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing pokemon file:", err.Error())
		return
	}
	fmt.Println("Pokemons saved:", len(pokemons))
}

// Generate a random coordinate within the grid
func getRandomCoord() Coord {
	return Coord{X: rand.Intn(GridSize), Y: rand.Intn(GridSize)}
}

// Spawn a wave of random pokemons
func pokemonSpawner() {
	for {
		spawnPokemons()
		time.Sleep(PokemonSpawnInterval)
	}
}

// Spawn a single random pokemon
func spawnPokemons() {
	for i := 0; i < PokemonPerWave; i++ {
		pokemonIDCounter++
		pokemon := generateRandomPokemon()
		pokemons = append(pokemons, pokemon)
		fmt.Println("Spawned pokemon:", pokemon.Name, "at", pokemon.Coord.X, pokemon.Coord.Y)
	}
	savePokemons()
}

// Initialize a new player
func initializePlayer(name string) *Player {
	playerIDCounter++
	player := &Player{
		ID:           playerIDCounter,
		Name:         name,
		PokemonList:  []Pokemon{},
		CurrentCoord: getRandomCoord(),
	}
	players = append(players, *player)
	savePlayers()
	fmt.Println("New player joined:", player.Name)
	return player
}

// Get player by name
func getPlayerByName(name string) *Player {
	for i, player := range players {
		if player.Name == name {
			return &players[i]
		}
	}
	return nil
}

// Move player to a new coordinate
func movePlayer(player *Player, newX, newY int) {
	if newX < 0 || newX >= GridSize || newY < 0 || newY >= GridSize {
		fmt.Println("Invalid move. Out of bounds.")
		return
	}

	player.mu.Lock()
	defer player.mu.Unlock()

	player.CurrentCoord.X = newX
	player.CurrentCoord.Y = newY
	fmt.Println("Player", player.Name, "moved to", newX, newY)
	savePlayers()
}

// Generate a random pokemon based on loaded pokemon data
func generateRandomPokemon() Pokemon {
	randCoord := getRandomCoord()
	pokemonIndex := rand.Intn(len(pokemons))
	pokemonData := pokemons[pokemonIndex]

	pokemonIDCounter++
	return Pokemon{
		ID:         pokemonIDCounter,
		Name:       pokemonData.Name,
		Type:       pokemonData.Type,
		BaseExp:    pokemonData.BaseExp,
		Speed:      pokemonData.Speed,
		Attack:     pokemonData.Attack,
		Defense:    pokemonData.Defense,
		SpecialAtk: pokemonData.SpecialAtk,
		SpecialDef: pokemonData.SpecialDef,
		HP:         pokemonData.HP,
		EV:         pokemonData.EV,
		CurrentExp: pokemonData.BaseExp,
		Level:      1,
		SpawnTime:  time.Now(),
		Coord:      randCoord,
	}
}

// Capture a pokemon at the player's current location
func capturePokemon(player *Player) {
	for i := len(pokemons) - 1; i >= 0; i-- {
		pokemon := &pokemons[i]
		if pokemon.Coord.X == player.CurrentCoord.X && pokemon.Coord.Y == player.CurrentCoord.Y {
			fmt.Println("Player", player.Name, "captured", pokemon.Name)
			player.mu.Lock()
			player.PokemonList = append(player.PokemonList, *pokemon)
			player.mu.Unlock()
			pokemons = append(pokemons[:i], pokemons[i+1:]...) // Remove captured pokemon
			savePokemons()
			savePlayers()
			return
		}
	}
	fmt.Println("No Pokemon to capture at", player.CurrentCoord.X, player.CurrentCoord.Y)
}
