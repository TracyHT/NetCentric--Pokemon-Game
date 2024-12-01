package player

type Player struct {
	ID       int               `json:"id"`
	Name     string            `json:"name"`
	Pokemons []CapturedPokemon `json:"pokemon_list"`
}

type CapturedPokemon struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Type       []string `json:"type"`
	BaseExp    int      `json:"base_exp"`
	HP         int      `json:"hp"`
	EV         float64  `json:"ev"`
	Level      int      `json:"level"`
	CurrentExp int      `json:"current_exp"`
	Speed      int      `json:"speed"`
	Attack     int      `json:"attack"`
	Defense    int      `json:"defense"`
	SpecialAtk int      `json:"special_atk"`
	SpecialDef int      `json:"special_def"`
}
