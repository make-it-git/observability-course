package app

// SearchService предоставляет функциональность поиска водителей.
type SearchService struct {
	locationManager *LocationManager
}

// NewSearchService создает новый SearchService.
func NewSearchService(locationManager *LocationManager) *SearchService {
	return &SearchService{
		locationManager: locationManager,
	}
}

// FindNearestDrivers ищет ближайших водителей к заданным координатам.
func (s *SearchService) FindNearestDrivers(lat, lon float64, limit int) []string {
	return s.locationManager.FindNearestDrivers(lat, lon, limit)
}
