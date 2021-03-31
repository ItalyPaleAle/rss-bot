package db

func V2() error {
	DB := GetDB()
	_, err := DB.Exec(`
ALTER TABLE feeds ADD COLUMN feed_title text not null default "";

UPDATE migrations SET version = 2 WHERE ROWID = 0;
`)
	if err != nil {
		return err
	}
	version = 2
	return nil
}
