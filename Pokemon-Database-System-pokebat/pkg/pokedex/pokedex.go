package pokedex

import (
	"encoding/json"
	"io/ioutil"
)

type Pokedex struct {
	Pokemons []Pokemon `json:"pokemons"`
}

type Pokemon struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	BaseExp    int     `json:"base_exp"`
	Speed      int     `json:"speed"`
	Attack     int     `json:"attack"`
	Defense    int     `json:"defense"`
	SpecialAtk int     `json:"special_atk"`
	SpecialDef int     `json:"special_def"`
	HP         int     `json:"hp"`
	EV         float64 `json:"ev"`
}

func LoadPokedex(filename string) (*Pokedex, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var pokedex Pokedex
	err = json.Unmarshal(data, &pokedex)
	if err != nil {
		return nil, err
	}

	return &pokedex, nil
}
