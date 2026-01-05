package services

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// BrazilStateCoordinates contém coordenadas aproximadas (capitais) de cada estado brasileiro
// Usado como fallback quando o técnico não tem localização real
var BrazilStateCoordinates = map[string]struct {
	Latitude  float64
	Longitude float64
	Capital   string
}{
	"AC": {Latitude: -9.9747, Longitude: -67.8249, Capital: "Rio Branco"},
	"AL": {Latitude: -9.6498, Longitude: -35.7089, Capital: "Maceió"},
	"AM": {Latitude: -3.1190, Longitude: -60.0217, Capital: "Manaus"},
	"AP": {Latitude: 0.0349, Longitude: -51.0694, Capital: "Macapá"},
	"BA": {Latitude: -12.9714, Longitude: -38.5014, Capital: "Salvador"},
	"CE": {Latitude: -3.7172, Longitude: -38.5433, Capital: "Fortaleza"},
	"DF": {Latitude: -15.7942, Longitude: -47.8822, Capital: "Brasília"},
	"ES": {Latitude: -20.3155, Longitude: -40.3128, Capital: "Vitória"},
	"GO": {Latitude: -16.6864, Longitude: -49.2643, Capital: "Goiânia"},
	"MA": {Latitude: -2.5307, Longitude: -44.3068, Capital: "São Luís"},
	"MG": {Latitude: -19.9167, Longitude: -43.9345, Capital: "Belo Horizonte"},
	"MS": {Latitude: -20.4697, Longitude: -54.6201, Capital: "Campo Grande"},
	"MT": {Latitude: -15.6014, Longitude: -56.0979, Capital: "Cuiabá"},
	"PA": {Latitude: -1.4558, Longitude: -48.4902, Capital: "Belém"},
	"PB": {Latitude: -7.1195, Longitude: -34.8450, Capital: "João Pessoa"},
	"PE": {Latitude: -8.0476, Longitude: -34.8770, Capital: "Recife"},
	"PI": {Latitude: -5.0892, Longitude: -42.8019, Capital: "Teresina"},
	"PR": {Latitude: -25.4284, Longitude: -49.2733, Capital: "Curitiba"},
	"RJ": {Latitude: -22.9068, Longitude: -43.1729, Capital: "Rio de Janeiro"},
	"RN": {Latitude: -5.7945, Longitude: -35.2110, Capital: "Natal"},
	"RO": {Latitude: -8.7612, Longitude: -63.9004, Capital: "Porto Velho"},
	"RR": {Latitude: 2.8235, Longitude: -60.6758, Capital: "Boa Vista"},
	"RS": {Latitude: -30.0346, Longitude: -51.2177, Capital: "Porto Alegre"},
	"SC": {Latitude: -27.5954, Longitude: -48.5480, Capital: "Florianópolis"},
	"SE": {Latitude: -10.9472, Longitude: -37.0731, Capital: "Aracaju"},
	"SP": {Latitude: -23.5505, Longitude: -46.6333, Capital: "São Paulo"},
	"TO": {Latitude: -10.1689, Longitude: -48.3317, Capital: "Palmas"},
}

// Principais cidades brasileiras com suas coordenadas
// Usado para melhor precisão quando temos o nome da cidade
var BrazilCityCoordinates = map[string]struct {
	Latitude  float64
	Longitude float64
}{
	// São Paulo - Interior e Grande SP
	"São Paulo":             {-23.5505, -46.6333},
	"Campinas":              {-22.9099, -47.0626},
	"Santos":                {-23.9608, -46.3336},
	"São José dos Campos":   {-23.2237, -45.9009},
	"Ribeirão Preto":        {-21.1767, -47.8208},
	"Sorocaba":              {-23.5015, -47.4526},
	"Santo André":           {-23.6737, -46.5432},
	"Osasco":                {-23.5324, -46.7917},
	"Guarulhos":             {-23.4538, -46.5333},
	"Barueri":               {-23.5105, -46.8768},
	"São Bernardo do Campo": {-23.6914, -46.5646},
	"São Caetano do Sul":    {-23.6229, -46.5548},
	"Mauá":                  {-23.6678, -46.4614},
	"Diadema":               {-23.6816, -46.6206},
	"Mogi das Cruzes":       {-23.5225, -46.1875},
	"Piracicaba":            {-22.7255, -47.6492},
	"Jundiaí":               {-23.1857, -46.8978},
	"Bauru":                 {-22.3246, -49.0871},
	"São José do Rio Preto": {-20.8113, -49.3758},
	"Franca":                {-20.5387, -47.4008},
	"Presidente Prudente":   {-22.1207, -51.3882},
	"Taubaté":               {-23.0225, -45.5556},
	"Limeira":               {-22.5640, -47.4013},
	"Marília":               {-22.2139, -49.9461},
	"Araraquara":            {-21.7948, -48.1756},
	"Jacareí":               {-23.2983, -45.9658},
	"Americana":             {-22.7373, -47.3300},
	"Sumaré":                {-22.8206, -47.2667},
	"Indaiatuba":            {-23.0877, -47.2181},
	"Itaquaquecetuba":       {-23.4862, -46.3486},
	"Praia Grande":          {-24.0058, -46.4022},
	"São Vicente":           {-23.9630, -46.3922},
	"Guarujá":               {-23.9932, -46.2564},
	"Cubatão":               {-23.8950, -46.4253},
	"Cotia":                 {-23.6039, -46.9189},
	"Taboão da Serra":       {-23.6217, -46.7582},
	"Embu das Artes":        {-23.6489, -46.8525},
	"Carapicuíba":           {-23.5224, -46.8406},
	"Itapevi":               {-23.5489, -46.9344},
	"Santana de Parnaíba":   {-23.4439, -46.9178},
	"Atibaia":               {-23.1164, -46.5561},
	"Bragança Paulista":     {-22.9517, -46.5419},
	"Mogi Mirim":            {-22.4319, -46.9578},
	"Mogi Guaçu":            {-22.3722, -46.9428},
	"Hortolândia":           {-22.8583, -47.2200},
	"Santa Bárbara d'Oeste": {-22.7542, -47.4142},
	"Valinhos":              {-22.9706, -46.9958},
	"Vinhedo":               {-23.0297, -46.9758},
	"Itatiba":               {-23.0058, -46.8389},
	"Paulínia":              {-22.7611, -47.1544},
	"Sertãozinho":           {-21.1378, -47.9900},
	"Jaboticabal":           {-21.2550, -48.3228},
	"Bebedouro":             {-20.9492, -48.4792},
	"Catanduva":             {-21.1378, -48.9728},
	"Votuporanga":           {-20.4217, -49.9728},
	"Fernandópolis":         {-20.2839, -50.2456},
	"Mirassol":              {-20.8197, -49.5208},
	"Assis":                 {-22.6617, -50.4122},
	"Ourinhos":              {-22.9789, -49.8706},
	"Botucatu":              {-22.8858, -48.4450},
	"Avaré":                 {-23.0992, -48.9264},
	"Itapetininga":          {-23.5917, -48.0531},
	"Registro":              {-24.4875, -47.8444},
	"Caraguatatuba":         {-23.6203, -45.4131},
	"São Sebastião":         {-23.8039, -45.4097},
	"Ubatuba":               {-23.4339, -45.0711},
	"Ilhabela":              {-23.7781, -45.3575},

	// Rio de Janeiro
	"Rio de Janeiro": {-22.9068, -43.1729},
	"Niterói":        {-22.8838, -43.1038},
	"Petrópolis":     {-22.5112, -43.1779},
	"Nova Iguaçu":    {-22.7556, -43.4503},

	// Minas Gerais
	"Belo Horizonte": {-19.9167, -43.9345},
	"Uberlândia":     {-18.9186, -48.2772},
	"Contagem":       {-19.9320, -44.0539},
	"Juiz de Fora":   {-21.7642, -43.3496},
	"Betim":          {-19.9681, -44.1985},

	// Bahia
	"Salvador":             {-12.9714, -38.5014},
	"Feira de Santana":     {-12.2667, -38.9667},
	"Vitória da Conquista": {-14.8619, -40.8389},

	// Paraná
	"Curitiba":     {-25.4284, -49.2733},
	"Londrina":     {-23.3045, -51.1696},
	"Maringá":      {-23.4273, -51.9375},
	"Ponta Grossa": {-25.0994, -50.1583},
	"Cascavel":     {-24.9578, -53.4595},

	// Rio Grande do Sul
	"Porto Alegre":  {-30.0346, -51.2177},
	"Caxias do Sul": {-29.1634, -51.1797},
	"Pelotas":       {-31.7654, -52.3376},
	"Canoas":        {-29.9178, -51.1839},

	// Santa Catarina
	"Florianópolis": {-27.5954, -48.5480},
	"Joinville":     {-26.3044, -48.8464},
	"Blumenau":      {-26.9194, -49.0661},
	"Chapecó":       {-27.0963, -52.6158},

	// Pernambuco
	"Recife":                  {-8.0476, -34.8770},
	"Olinda":                  {-8.0089, -34.8553},
	"Jaboatão dos Guararapes": {-8.1129, -35.0154},
	"Caruaru":                 {-8.2760, -35.9761},

	// Ceará
	"Fortaleza":         {-3.7172, -38.5433},
	"Caucaia":           {-3.7361, -38.6531},
	"Juazeiro do Norte": {-7.2130, -39.3151},

	// Amazonas
	"Manaus": {-3.1190, -60.0217},

	// Pará
	"Belém":      {-1.4558, -48.4902},
	"Ananindeua": {-1.3659, -48.3726},
	"Santarém":   {-2.4430, -54.7066},

	// Goiás
	"Goiânia":              {-16.6864, -49.2643},
	"Aparecida de Goiânia": {-16.8232, -49.2469},
	"Anápolis":             {-16.3281, -48.9530},

	// Distrito Federal
	"Brasília": {-15.7942, -47.8822},

	// Maranhão
	"São Luís":   {-2.5307, -44.3068},
	"Imperatriz": {-5.5184, -47.4777},

	// Rio Grande do Norte
	"Natal":   {-5.7945, -35.2110},
	"Mossoró": {-5.1878, -37.3441},

	// Paraíba
	"João Pessoa":    {-7.1195, -34.8450},
	"Campina Grande": {-7.2300, -35.8810},

	// Alagoas
	"Maceió": {-9.6498, -35.7089},

	// Piauí
	"Teresina": {-5.0892, -42.8019},

	// Sergipe
	"Aracaju": {-10.9472, -37.0731},

	// Mato Grosso
	"Cuiabá":        {-15.6014, -56.0979},
	"Várzea Grande": {-15.6470, -56.1322},
	"Rondonópolis":  {-16.4673, -54.6372},

	// Mato Grosso do Sul
	"Campo Grande": {-20.4697, -54.6201},
	"Dourados":     {-22.2211, -54.8056},
	"Três Lagoas":  {-20.7847, -51.7010},

	// Espírito Santo
	"Vitória":    {-20.3155, -40.3128},
	"Vila Velha": {-20.3297, -40.2925},
	"Serra":      {-20.1286, -40.3075},
	"Cariacica":  {-20.2636, -40.4164},

	// Tocantins
	"Palmas":    {-10.1689, -48.3317},
	"Araguaína": {-7.1920, -48.2077},

	// Rondônia
	"Porto Velho": {-8.7612, -63.9004},
	"Ji-Paraná":   {-10.8853, -61.9517},

	// Acre
	"Rio Branco": {-9.9747, -67.8249},

	// Amapá
	"Macapá": {-0.0349, -51.0694},

	// Roraima
	"Boa Vista": {2.8235, -60.6758},
}

// normalizeCity normaliza o nome da cidade para busca
// Remove acentos, converte para minúsculas, remove espaços extras
func normalizeCity(city string) string {
	// Remove espaços extras e converte para minúsculas
	city = strings.TrimSpace(strings.ToLower(city))

	// Remove acentos
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, city)

	return result
}

// buildNormalizedCityMap cria um mapa normalizado das cidades
var normalizedCityMap = func() map[string]struct {
	Latitude  float64
	Longitude float64
} {
	result := make(map[string]struct {
		Latitude  float64
		Longitude float64
	})

	for city, coords := range BrazilCityCoordinates {
		normalized := normalizeCity(city)
		result[normalized] = coords
	}

	return result
}()

// GetCoordinatesForLocation retorna coordenadas para uma cidade/estado
// Prioriza a cidade se disponível, senão usa a capital do estado
func GetCoordinatesForLocation(city, state string) (lat, lng float64, hasExact bool) {
	// Tenta primeiro pela cidade (nome exato)
	if city != "" {
		if coords, ok := BrazilCityCoordinates[city]; ok {
			return coords.Latitude, coords.Longitude, true
		}

		// Tenta com nome normalizado
		normalizedCity := normalizeCity(city)
		if coords, ok := normalizedCityMap[normalizedCity]; ok {
			return coords.Latitude, coords.Longitude, true
		}
	}

	// Fallback para capital do estado
	if state != "" {
		if coords, ok := BrazilStateCoordinates[state]; ok {
			return coords.Latitude, coords.Longitude, false
		}
	}

	// Fallback para centro do Brasil (Brasília)
	return -15.7942, -47.8822, false
}

// Legacy function - kept for compatibility but now just calls GetCoordinatesForLocation
func GetCoordinatesForLocationWithOffset(city, state, technicianID string) (lat, lng float64, hasExact bool) {
	return GetCoordinatesForLocation(city, state)
}


