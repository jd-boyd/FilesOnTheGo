// Package migrations provides database migration scripts for FilesOnTheGo.
// It creates the necessary collections and indexes for the application.
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
)

// pointer is a helper function to create string pointers
func pointer(s string) *string {
	return &s
}

// InitialSchema creates all collections for FilesOnTheGo
func init() {
	core.AppMigrations.Register(func(txApp core.App) error {
		// 1. Extend users collection (superusers) with custom fields
		usersCollection, err := txApp.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return err
		}

		// Add storage_quota field (default: 100GB = 107374182400 bytes)
		usersCollection.Fields.Add(&core.NumberField{
			Name:     "storage_quota",
			Required: false,
		})

		// Add storage_used field (default: 0)
		usersCollection.Fields.Add(&core.NumberField{
			Name:     "storage_used",
			Required: false,
		})

		// Add is_admin field (default: false)
		usersCollection.Fields.Add(&core.BoolField{
			Name:     "is_admin",
			Required: false,
		})

		if err := txApp.Save(usersCollection); err != nil {
			return err
		}

		// 2. Create Directories collection
		directoriesCollection := core.NewBaseCollection("directories")
		directoriesCollection.ListRule = pointer("@request.auth.id = user.id")
		directoriesCollection.ViewRule = pointer("@request.auth.id = user.id")
		directoriesCollection.CreateRule = pointer("@request.auth.id = user.id")
		directoriesCollection.UpdateRule = pointer("@request.auth.id = user.id")
		directoriesCollection.DeleteRule = pointer("@request.auth.id = user.id")

		directoriesCollection.Fields.Add(
			&core.TextField{
				Name:     "name",
				Required: true,
				Max:      255,
			},
		)

		directoriesCollection.Fields.Add(
			&core.TextField{
				Name:     "path",
				Required: true,
				Max:      1024,
			},
		)

		directoriesCollection.Fields.Add(
			&core.RelationField{
				Name:          "user",
				Required:      true,
				CollectionId:  usersCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		// Create indexes for directories (excluding parent_directory for now)
		directoriesCollection.Indexes = []string{
			"CREATE INDEX idx_directories_user_path ON directories (user, path)",
			"CREATE INDEX idx_directories_path ON directories (path)",
		}

		// First save without the self-referential field
		if err := txApp.Save(directoriesCollection); err != nil {
			return err
		}

		// Now add the self-referential parent_directory field with the collection ID
		directoriesCollection.Fields.Add(
			&core.RelationField{
				Name:          "parent_directory",
				Required:      false,
				CollectionId:  directoriesCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		// Add index for parent_directory
		directoriesCollection.Indexes = append(directoriesCollection.Indexes,
			"CREATE INDEX idx_directories_user_parent ON directories (user, parent_directory)",
		)

		if err := txApp.Save(directoriesCollection); err != nil {
			return err
		}

		// 3. Create Files collection
		filesCollection := core.NewBaseCollection("files")
		filesCollection.ListRule = pointer("@request.auth.id = user.id")
		filesCollection.ViewRule = pointer("@request.auth.id = user.id")
		filesCollection.CreateRule = pointer("@request.auth.id = user.id")
		filesCollection.UpdateRule = pointer("@request.auth.id = user.id")
		filesCollection.DeleteRule = pointer("@request.auth.id = user.id")

		filesCollection.Fields.Add(
			&core.TextField{
				Name:     "name",
				Required: true,
				Max:      255,
			},
		)

		filesCollection.Fields.Add(
			&core.TextField{
				Name:     "path",
				Required: true,
				Max:      1024,
			},
		)

		filesCollection.Fields.Add(
			&core.RelationField{
				Name:          "user",
				Required:      true,
				CollectionId:  usersCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		filesCollection.Fields.Add(
			&core.RelationField{
				Name:          "parent_directory",
				Required:      false,
				CollectionId:  directoriesCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		filesCollection.Fields.Add(
			&core.NumberField{
				Name:     "size",
				Required: true,
			},
		)

		filesCollection.Fields.Add(
			&core.TextField{
				Name:     "mime_type",
				Required: false,
				Max:      255,
			},
		)

		filesCollection.Fields.Add(
			&core.TextField{
				Name:     "s3_key",
				Required: true,
				Max:      512,
			},
		)

		filesCollection.Fields.Add(
			&core.TextField{
				Name:     "s3_bucket",
				Required: true,
				Max:      255,
			},
		)

		filesCollection.Fields.Add(
			&core.TextField{
				Name:     "checksum",
				Required: false,
				Max:      128,
			},
		)

		// Create indexes for files
		filesCollection.Indexes = []string{
			"CREATE UNIQUE INDEX idx_files_s3_key ON files (s3_key)",
			"CREATE INDEX idx_files_user_path ON files (user, path)",
			"CREATE INDEX idx_files_user_parent ON files (user, parent_directory)",
		}

		if err := txApp.Save(filesCollection); err != nil {
			return err
		}

		// 4. Create Shares collection
		sharesCollection := core.NewBaseCollection("shares")
		sharesCollection.ListRule = pointer("@request.auth.id = user.id")
		sharesCollection.ViewRule = pointer("@request.auth.id = user.id")
		sharesCollection.CreateRule = pointer("@request.auth.id = user.id")
		sharesCollection.UpdateRule = pointer("@request.auth.id = user.id")
		sharesCollection.DeleteRule = pointer("@request.auth.id = user.id")

		sharesCollection.Fields.Add(
			&core.RelationField{
				Name:          "user",
				Required:      true,
				CollectionId:  usersCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		sharesCollection.Fields.Add(
			&core.SelectField{
				Name:      "resource_type",
				Required:  true,
				MaxSelect: 1,
				Values:    []string{"file", "directory"},
			},
		)

		sharesCollection.Fields.Add(
			&core.RelationField{
				Name:          "file",
				Required:      false,
				CollectionId:  filesCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		sharesCollection.Fields.Add(
			&core.RelationField{
				Name:          "directory",
				Required:      false,
				CollectionId:  directoriesCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		sharesCollection.Fields.Add(
			&core.TextField{
				Name:     "share_token",
				Required: true,
				Max:      128,
			},
		)

		sharesCollection.Fields.Add(
			&core.SelectField{
				Name:      "permission_type",
				Required:  true,
				MaxSelect: 1,
				Values:    []string{"read", "read_upload", "upload_only"},
			},
		)

		sharesCollection.Fields.Add(
			&core.TextField{
				Name:     "password_hash",
				Required: false,
				Max:      255,
			},
		)

		sharesCollection.Fields.Add(
			&core.DateField{
				Name:     "expires_at",
				Required: false,
			},
		)

		sharesCollection.Fields.Add(
			&core.NumberField{
				Name:     "access_count",
				Required: false,
			},
		)

		// Create indexes for shares
		sharesCollection.Indexes = []string{
			"CREATE UNIQUE INDEX idx_shares_token ON shares (share_token)",
			"CREATE INDEX idx_shares_user ON shares (user)",
			"CREATE INDEX idx_shares_expires ON shares (expires_at)",
		}

		if err := txApp.Save(sharesCollection); err != nil {
			return err
		}

		// 5. Create Share Access Logs collection
		shareAccessLogsCollection := core.NewBaseCollection("share_access_logs")
		shareAccessLogsCollection.ListRule = pointer("@request.auth.id = share.user.id")
		shareAccessLogsCollection.ViewRule = pointer("@request.auth.id = share.user.id")
		shareAccessLogsCollection.CreateRule = pointer("") // Allow any (backend only)
		shareAccessLogsCollection.UpdateRule = pointer("") // Deny
		shareAccessLogsCollection.DeleteRule = pointer("") // Deny

		shareAccessLogsCollection.Fields.Add(
			&core.RelationField{
				Name:          "share",
				Required:      true,
				CollectionId:  sharesCollection.Id,
				CascadeDelete: true,
				MaxSelect:     1,
			},
		)

		shareAccessLogsCollection.Fields.Add(
			&core.TextField{
				Name:     "ip_address",
				Required: false,
				Max:      45, // IPv6 max length
			},
		)

		shareAccessLogsCollection.Fields.Add(
			&core.TextField{
				Name:     "user_agent",
				Required: false,
				Max:      512,
			},
		)

		shareAccessLogsCollection.Fields.Add(
			&core.SelectField{
				Name:      "action",
				Required:  true,
				MaxSelect: 1,
				Values:    []string{"view", "download", "upload"},
			},
		)

		shareAccessLogsCollection.Fields.Add(
			&core.TextField{
				Name:     "file_name",
				Required: false,
				Max:      255,
			},
		)

		shareAccessLogsCollection.Fields.Add(
			&core.DateField{
				Name:     "accessed_at",
				Required: false,
			},
		)

		// Create indexes for share access logs
		shareAccessLogsCollection.Indexes = []string{
			"CREATE INDEX idx_share_logs_share_accessed ON share_access_logs (share, accessed_at)",
		}

		if err := txApp.Save(shareAccessLogsCollection); err != nil {
			return err
		}

		return nil
	}, func(txApp core.App) error {
		// Down migration - drop all collections
		collections := []string{
			"share_access_logs",
			"shares",
			"files",
			"directories",
		}

		for _, name := range collections {
			collection, err := txApp.FindCollectionByNameOrId(name)
			if err != nil {
				continue // Collection might not exist
			}
			if err := txApp.Delete(collection); err != nil {
				return err
			}
		}

		// Remove custom fields from users collection
		usersCollection, err := txApp.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return err
		}

		// Remove added fields
		usersCollection.Fields.RemoveByName("storage_quota")
		usersCollection.Fields.RemoveByName("storage_used")
		usersCollection.Fields.RemoveByName("is_admin")

		if err := txApp.Save(usersCollection); err != nil {
			return err
		}

		return nil
	})
}
