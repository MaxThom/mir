package surreal

import "github.com/surrealdb/surrealdb.go"

func ConnectToDb(url, namespace, database, user, password string) (*surrealdb.DB, error) {
	db, err := surrealdb.New(url)
	if err != nil {
		return nil, err
	}

	if _, err = db.Signin(map[string]any{
		"user": user,
		"pass": password,
	}); err != nil {
		return nil, err
	}

	if _, err = db.Use(namespace, database); err != nil {
		return nil, err
	}

	return db, nil
}

func CreateUpdateQuery(fields map[string]interface{}) error {
	return nil
}

// Create ReadQuery
func CreateReadQuery(fields []string) error {
	return nil
}
