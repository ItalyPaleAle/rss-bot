package db

func V1() error {
	DB := GetDB()
	_, err := DB.Exec(`
CREATE TABLE webhooks (
	webhook_id text not null,
	webhook_key blob not null,
	webhook_created integer not null,
	chat_id integer not null
);

CREATE UNIQUE INDEX webhooks_webhook_id ON webhooks (webhook_id);
CREATE INDEX webhooks_webhook_created ON webhooks (webhook_created);

INSERT INTO migrations (ROWID, version) VALUES (0, 1);
`)
	if err != nil {
		return err
	}
	version = 1
	return nil
}
