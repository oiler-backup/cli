package cmd

func setupFlags() {
	backupCreateCmd.Flags().StringVar(&db, "db", "", "DB specification in the format dbType@dbUri:dbPort/dbName")
	backupCreateCmd.Flags().StringVar(&dbUser, "db-user", "", "DB user")
	backupCreateCmd.Flags().StringVar(&dbPass, "db-pass", "", "DB password")
	backupCreateCmd.Flags().BoolVar(&dbUserStdin, "db-user-stdin", false, "Prompt for DB user from stdin")
	backupCreateCmd.Flags().BoolVar(&dbPassStdin, "db-pass-stdin", false, "Prompt for DB password from stdin")
	backupCreateCmd.Flags().StringVar(&s3, "s3", "", "S3 specification in the format endpoint:port/bucket")
	backupCreateCmd.Flags().StringVar(&s3AccessKey, "s3-access-key", "", "S3 access key")
	backupCreateCmd.Flags().StringVar(&s3SecretKey, "s3-secret-key", "", "S3 secret key")
	backupCreateCmd.Flags().BoolVar(&s3AccessKeyStdin, "s3-access-key-stdin", false, "Prompt for S3 access key from stdin")
	backupCreateCmd.Flags().BoolVar(&s3SecretKeyStdin, "s3-secret-key-stdin", false, "Prompt for S3 secret key from stdin")
	backupCreateCmd.Flags().StringVar(&schedule, "schedule", "*/1 * * * *", "Cron schedule for backups")
	backupCreateCmd.Flags().Int64Var(&maxBackupCount, "max-backup-count", 2, "Maximum number of backups to retain")
	backupCreateCmd.Flags().StringVar(&backupRequestName, "name", "", "Name of the BackupRequest")
	backupCreateCmd.MarkFlagRequired("name")
	backupCreateCmd.MarkFlagRequired("db")
	backupCreateCmd.MarkFlagRequired("s3")
}
