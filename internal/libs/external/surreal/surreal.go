package surreal

import "github.com/maxthom/surrealdb.go"

func ConnectToDb(url, namespace, database, user, password string) (*surrealdb.DB, error) {
	db, err := surrealdb.New(url)
	if err != nil {
		return nil, err
	}

	if _, err = db.SignIn(&surrealdb.Auth{
		Username: user,
		Password: password,
	}); err != nil {
		return nil, err
	}

	if err = db.Use(namespace, database); err != nil {
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
