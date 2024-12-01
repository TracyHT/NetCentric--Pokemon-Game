package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
}

func FetchPokemonData(url string) ([]Pokemon, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var pokemons []Pokemon

	// Iterate over each list item in the list containing Pokémon data
	doc.Find(".infocard").Each(func(i int, s *goquery.Selection) {
		if len(pokemons) > 200 {
			return
		}
		var pokemon Pokemon

		// Extract Pokémon ID
		idStr, _ := s.Find(".infocard-cell-data").First().Attr("data-sprite")
		id, _ := strconv.Atoi(strings.TrimPrefix(idStr, "/sprites/"))
		pokemon.ID = id

		// Extract Pokémon name
		pokemon.Name = strings.TrimSpace(s.Find(".ent-name").Text())

		// Extract Pokémon type(s)
		s.Find(".itype").Each(func(i int, s *goquery.Selection) {
			pokemon.Type = append(pokemon.Type, strings.TrimSpace(s.Text()))
		})

		// Extract Pokémon detail page URL
		detailURL, _ := s.Find("a").First().Attr("href")
		fullDetailURL := "https://pokemondb.net" + detailURL

		// Fetch Pokémon details from the detail page
		if err := FetchPokemonDetails(fullDetailURL, &pokemon); err != nil {
			fmt.Printf("Error fetching details for %s: %v\n", pokemon.Name, err)
			return
		}

		// Append Pokémon to the list
		pokemons = append(pokemons, pokemon)
	})

	return pokemons, nil
}

func FetchPokemonDetails(url string, pokemon *Pokemon) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// Variables to track if required data is found
	var baseExpFound, evFound, statsFound bool

	// Find all vitals tables
	doc.Find(".vitals-table").Each(func(i int, s *goquery.Selection) {
		// Fetch Base Exp, EV Yield, and stats
		s.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
			attrName := strings.TrimSpace(row.Find("th").Text())
			attrValue := strings.TrimSpace(row.Find("td").Text())

			// Check for base experience, EV yield, and stats
			switch attrName {
			case "Base Exp.":
				pokemon.BaseExp, _ = strconv.Atoi(attrValue)
				baseExpFound = true
			case "EV yield":
				evStr := strings.Fields(attrValue)[0]
				pokemon.EV, _ = strconv.ParseFloat(evStr, 64)
				evFound = true
			default:
				// Check if attrValue contains stats
				statParts := strings.Split(attrValue, "\n")
				if len(statParts) == 3 {
					statValue, _ := strconv.Atoi(statParts[0])
					statsFound = true
					switch attrName {
					case "HP":
						pokemon.HP = statValue
					case "Attack":
						pokemon.Attack = statValue
					case "Defense":
						pokemon.Defense = statValue
					case "Sp. Atk":
						pokemon.SpecialAtk = statValue
					case "Sp. Def":
						pokemon.SpecialDef = statValue
					case "Speed":
						pokemon.Speed = statValue
					}
				}
			}
		})

		// Check if stats are found
		if statsFound {
			return
		}
	})

	// Ensure all required data is fetched
	if !baseExpFound || !evFound || !statsFound {
		return fmt.Errorf("not all required data fetched")
	}

	return nil
}

func main() {
	pokemonDataURL := "https://pokemondb.net/pokedex/national"

	pokemons, err := FetchPokemonData(pokemonDataURL)
	if err != nil {
		fmt.Println("Error fetching Pokémon data:", err)
		return
	}

	file, err := json.MarshalIndent(pokemons, "", "  ")
	if err != nil {
		fmt.Println("Error encoding Pokémon data:", err)
		return
	}

	err = ioutil.WriteFile("pokedex.json", file, 0644)
	if err != nil {
		fmt.Println("Error saving Pokémon data to file:", err)
		return
	}

	fmt.Println("Pokédex data has been successfully saved.")
}
