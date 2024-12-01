package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"pokemon-database-system-pokebat/pkg/player"
	"strconv"
	"strings"
	"sync"
	"time"
)

type gamer struct {
	name        string
	fighterList map[int]player.CapturedPokemon
	fighter     player.CapturedPokemon
	addr        *net.UDPAddr
}

var (
	players      = make(map[string]player.Player)
	gamers       = make(map[string]*gamer) // Store pointers to gamers
	mutex        sync.Mutex
	battleActive bool
	serverConn   *net.UDPConn // Global server UDP connection
	divider      = "-----------------------"
)

func printPokemonInfo(pokemon player.CapturedPokemon) string {
	info := fmt.Sprintf("%d. Name: %s | ", pokemon.ID, pokemon.Name)
	info += fmt.Sprintf("Type: %v | ", pokemon.Type)
	info += fmt.Sprintf("Base Exp: %d |", pokemon.BaseExp)
	info += fmt.Sprintf("HP: %d | ", pokemon.HP)
	info += fmt.Sprintf("EV: %.1f | ", pokemon.EV)
	info += fmt.Sprintf("Level: %d | ", pokemon.Level)
	info += fmt.Sprintf("Current Exp: %d\n", pokemon.CurrentExp)
	info += fmt.Sprintf("Speed: %d | ", pokemon.Speed)
	info += fmt.Sprintf("Attack: %d | ", pokemon.Attack)
	info += fmt.Sprintf("Defense: %d | ", pokemon.Defense)
	info += fmt.Sprintf("Special Atk: %d | ", pokemon.SpecialAtk)
	info += fmt.Sprintf("Special Def: %d\n", pokemon.SpecialDef)
	info += "\n"
	return info
}

func choosePokemon(g gamer, p player.Player, conn *net.UDPConn) []player.CapturedPokemon {
	var chosenPokemons []player.CapturedPokemon

	sendMessage(conn, g.addr, fmt.Sprintf("Player: %s\n", g.name))
	for _, pokemon := range p.Pokemons {
		sendMessage(conn, g.addr, printPokemonInfo(pokemon))
	}

	sendMessage(conn, g.addr, "Choose 3 Pokémon (enter the pokemon ids separated by spaces): ")

	message := receiveMessage(conn)
	fmt.Println("Received message:", message) // Debugging statement
	choices := parseInput(message, 3)

	for _, choice := range choices {
		chosenPokemons = append(chosenPokemons, p.Pokemons[choice-1])
	}

	fmt.Println(chosenPokemons)

	return chosenPokemons
}

func parseInput(input string, maxChoices int) []int {
	choices := []int{}
	for _, char := range input {
		if char != ' ' {
			choice, err := strconv.Atoi(string(char))
			if err != nil || choice < 1 || choice > maxChoices {
				continue
			}
			choices = append(choices, choice)
		}
	}
	return choices
}

func switchTurn(attacker, defender *gamer) {
	*attacker, *defender = *defender, *attacker
}

func wait(i int) {
	time.Sleep(time.Duration(i) * time.Second)
}

func selectFighter(g *gamer, conn *net.UDPConn) bool {
	// Check if there are any available fighters
	available := false
	for _, pokemon := range g.fighterList {
		if pokemon.HP > 0 {
			available = true
			sendMessage(conn, g.addr, printPokemonInfo(pokemon))
		}
	}

	if !available {
		sendMessage(conn, g.addr, "You have no available fighters left!")
		return false
	}

	for {
		sendMessage(conn, g.addr, "Select your fighter by ID: ")
		input := receiveMessage(conn)
		id, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			sendMessage(conn, g.addr, "Invalid input. Please enter a valid Pokemon ID.")
			continue
		}

		selectedPokemon, ok := g.fighterList[id]
		if !ok || selectedPokemon.HP <= 0 {
			sendMessage(conn, g.addr, "Invalid selection or selected Pokemon has fainted.")
			continue
		}

		g.fighter = selectedPokemon
		sendMessage(conn, g.addr, fmt.Sprintf("Selected fighter: %s\n", g.fighter.Name))
		return true
	}
}

func startBattle(gamer1, gamer2 *gamer) {
	fmt.Println("Battle begins!")

	sendMessage(serverConn, gamer1.addr, "Two players connected. Starting the battle!\n")
	sendMessage(serverConn, gamer2.addr, "Two players connected. Starting the battle!\n")

	var attacker, defender *gamer
	if gamer1.fighter.Speed >= gamer2.fighter.Speed {
		attacker = gamer1
		defender = gamer2
	} else {
		attacker = gamer2
		defender = gamer1
	}

	i := 0
	for {
		i++
		sendMessage(serverConn, defender.addr, divider)
		sendMessage(serverConn, attacker.addr, divider)
		sendMessage(serverConn, attacker.addr, fmt.Sprintf("Turn %d, attacker %s: \n", i, attacker.name))
		sendMessage(serverConn, defender.addr, fmt.Sprintf("Turn %d, attacker %s: \n", i, attacker.name))

		attack(attacker, defender)

		sendMessage(serverConn, attacker.addr, "Turn result: ")
		sendMessage(serverConn, attacker.addr, printPokemonInfo(defender.fighter))

		sendMessage(serverConn, defender.addr, "Turn result: ")
		sendMessage(serverConn, defender.addr, printPokemonInfo(defender.fighter))

		sendMessage(serverConn, defender.addr, divider)
		sendMessage(serverConn, attacker.addr, divider)

		if defender.fighter.HP <= 0 {
			sendMessage(serverConn, defender.addr, divider)
			sendMessage(serverConn, attacker.addr, divider)
			sendMessage(serverConn, defender.addr, fmt.Sprintf("%s's %s fainted!\n, you have to switch your fighter!", defender.name, defender.fighter.Name))
			sendMessage(serverConn, attacker.addr, "The opponent's fighter is fainted!, wait for them to switch the fighter!")

			status := selectFighter(defender, serverConn)
			if !status {
				//attacker win
				distributeExperiencePoints(attacker, defender, serverConn)
				sendMessage(serverConn, defender.addr, "You lose!")
				sendMessage(serverConn, attacker.addr, "You win!")
				break
			}
		}

		sendMessage(serverConn, attacker.addr, fmt.Sprintf("%s, do you want to switch your fighter?(Y/N)", attacker.name))

		switchTurn(attacker, defender)
	}

	sendMessage(serverConn, attacker.addr, "Battle ended!")
}

func distributeExperiencePoints(winner, loser *gamer, conn *net.UDPConn) {
	totalExp := 0

	// Calculate total accumulated experience points from the loser's team
	for _, pokemon := range loser.fighterList {
		totalExp += pokemon.CurrentExp
	}

	// Calculate 1/3 of the total accumulated experience points

	expPerPokemon := totalExp / (3 * len(loser.fighterList))
	sendMessage(conn, winner.addr, fmt.Sprintf("Each pokemon of %s will have %d bonus exp!", winner.name, expPerPokemon))
	// Distribute experience points to each Pokémon on the winner's team
	for _, pokemon := range winner.fighterList {
		pokemon.CurrentExp += expPerPokemon
		// Ensure CurrentExp does not exceed BaseExp
		if pokemon.CurrentExp > pokemon.BaseExp {
			pokemon.CurrentExp = pokemon.BaseExp
		}
	}
}

func attack(attacker, defender *gamer) {
	// Determine if it's a special attack or not
	rand.Seed(time.Now().UnixNano())
	isSpecialAttack := rand.Intn(2) == 0

	// Print attacking and defending Pokémon information
	sendMessage(serverConn, attacker.addr, "Attacking: ")
	sendMessage(serverConn, attacker.addr, printPokemonInfo(attacker.fighter))
	sendMessage(serverConn, defender.addr, "Attacking: ")
	sendMessage(serverConn, defender.addr, printPokemonInfo(attacker.fighter))
	wait(2)

	sendMessage(serverConn, defender.addr, "Defending: ")
	sendMessage(serverConn, defender.addr, printPokemonInfo(defender.fighter))
	sendMessage(serverConn, attacker.addr, "Defending: ")
	sendMessage(serverConn, attacker.addr, printPokemonInfo(defender.fighter))
	wait(2)

	// Calculate damage based on attack types
	var damage int
	if isSpecialAttack {
		damage = int(float64(attacker.fighter.SpecialAtk) - float64(defender.fighter.SpecialDef))
	} else {
		damage = attacker.fighter.Attack - defender.fighter.Defense
	}

	// Ensure damage is at least 1
	if damage < 1 {
		damage = 1
	}

	// Apply damage to the defender's HP
	defender.fighter.HP -= damage

	// Update defender's fighterList entry with the new HP
	defender.fighterList[defender.fighter.ID] = defender.fighter

	// Inform players about the attack and damage dealt
	if isSpecialAttack {
		sendMessage(serverConn, attacker.addr, fmt.Sprintf("%s used a special attack!\n", attacker.fighter.Name))
		sendMessage(serverConn, defender.addr, fmt.Sprintf("%s used a special attack!\n", attacker.fighter.Name))
	} else {
		sendMessage(serverConn, attacker.addr, fmt.Sprintf("%s used a normal attack!\n", attacker.fighter.Name))
		sendMessage(serverConn, defender.addr, fmt.Sprintf("%s used a normal attack!\n", attacker.fighter.Name))
	}

	sendMessage(serverConn, attacker.addr, divider)
	sendMessage(serverConn, defender.addr, divider)
	sendMessage(serverConn, attacker.addr, fmt.Sprintf("Damage dealt: %d\n", damage))
	sendMessage(serverConn, attacker.addr, fmt.Sprintf("%s's HP: %d\n", defender.fighter.Name, defender.fighter.HP))
	sendMessage(serverConn, defender.addr, fmt.Sprintf("Damage dealt: %d\n", damage))
	sendMessage(serverConn, defender.addr, fmt.Sprintf("%s's HP: %d\n", defender.fighter.Name, defender.fighter.HP))

	wait(3)
	// Prompt the attacker to switch their fighter
	sendMessage(serverConn, attacker.addr, fmt.Sprintf("%s, do you want to switch your fighter? (Y/N)", attacker.name))
	response := receiveMessage(serverConn)
	if strings.TrimSpace(strings.ToUpper(response)) == "Y" {
		if selectFighter(attacker, serverConn) {
			sendMessage(serverConn, attacker.addr, "You have switched your fighter.")
		} else {
			sendMessage(serverConn, attacker.addr, "Failed to switch fighter. Continue with the current fighter.")
		}
	}

	//wait(2)
}

func sendMessage(conn *net.UDPConn, addr *net.UDPAddr, message string) {
	conn.WriteToUDP([]byte(message), addr)
}

func receiveMessage(conn *net.UDPConn) string {
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error reading from UDP:", err)
		return ""
	}
	return string(buffer[:n])
}

func handleClient(conn *net.UDPConn, addr *net.UDPAddr, playerName string) {

	mutex.Lock()
	defer mutex.Unlock()

	p, exists := players[playerName]
	if !exists {
		sendMessage(conn, addr, "FAILED: no player found "+playerName)
		fmt.Println("Error: Player not found:", playerName)
		return
	}

	g := &gamer{
		name:        playerName,
		fighterList: make(map[int]player.CapturedPokemon),
		addr:        addr,
	}

	chosenPokemons := choosePokemon(*g, p, conn)
	for _, pokemon := range chosenPokemons {
		g.fighterList[pokemon.ID] = pokemon
	}

	g.fighter = chosenPokemons[0]

	gamers[playerName] = g
	sendMessage(conn, addr, "SUCCESS: You have registered as "+playerName)
	fmt.Println("Player registered:", playerName)
}

func handleMessage(msg string, senderAddr *net.UDPAddr) {
	// Handle any other messages here
}

func handleLogin(conn *net.UDPConn, addr *net.UDPAddr, msg string) {
	mutex.Lock()
	defer mutex.Unlock()

	username := strings.Split(msg, ":")[1]
	if _, exists := players[username]; exists {
		// Username already exists, notify the client
		successMessage := "SUCCESS: You have registered as " + username
		sendMessage(conn, addr, successMessage)
		fmt.Println("Success message sent to", addr, ":", successMessage)
		sendMessage(conn, addr, "Welcome to the Pokemon Battle Server!\n")
		go handleClient(conn, addr, username)
	} else {
		// Notify the client that no player found with this name
		errorMessage := "FAILED: no player found " + username
		sendMessage(conn, addr, errorMessage)
		fmt.Println("Error message sent to", addr, ":", errorMessage)
	}
}

func handleLogout(msg string) {
	mutex.Lock()
	defer mutex.Unlock()

	username := strings.Split(msg, ":")[1]
	delete(players, username)
	fmt.Println("Client", username, "logged out")
}

func loadPlayerData() {
	file, err := os.Open("../../player.json")
	if err != nil {
		fmt.Println("Error opening player data file:", err)
		return
	}
	defer file.Close()

	var playersData []player.Player

	if err := json.NewDecoder(file).Decode(&playersData); err != nil {
		fmt.Println("Error decoding player data:", err)
		return
	}
	for _, pd := range playersData {
		players[pd.Name] = pd
	}

	fmt.Printf("Loaded %d players\n", len(players))
}

func main() {
	loadPlayerData()

	// Create UDP address
	serverAddr, _ := net.ResolveUDPAddr("udp", ":8080")

	// Create UDP listener
	serverConn, _ = net.ListenUDP("udp", serverAddr)
	defer serverConn.Close()

	fmt.Println("Server started, waiting for players...")

	// Buffer for incoming messages
	buf := make([]byte, 1024)

	for {
		n, addr, _ := serverConn.ReadFromUDP(buf)
		msg := string(buf[:n])
		fmt.Println("Received message from client:", msg)

		if strings.HasPrefix(msg, "LOGIN:") {
			// Handle login message
			handleLogin(serverConn, addr, msg)
		} else if strings.HasPrefix(msg, "LOGOUT:") {
			// Handle logout message
			handleLogout(msg)
		} else {
			// Handle other messages
			mutex.Lock()
			if len(gamers) == 2 && !battleActive {
				battleActive = true
				var gamer1, gamer2 *gamer
				for _, g := range gamers {
					if gamer1 == nil {
						gamer1 = g
					} else {
						gamer2 = g
					}
				}
				mutex.Unlock()

				go startBattle(gamer1, gamer2)

				mutex.Lock()
				gamers = make(map[string]*gamer)
				battleActive = false
				mutex.Unlock()
			} else {
				mutex.Unlock()
			}
			go handleMessage(msg, addr)
		}
	}
}
