//go:build !unit_test

package tenant_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/goto/optimus/core/tenant"
	postgres "github.com/goto/optimus/internal/store/postgres/tenant"
	"github.com/goto/optimus/tests/setup"
)

func TestPostgresSecretRepository(t *testing.T) {
	ctx := context.Background()

	proj, _ := tenant.NewProject("test-proj",
		map[string]string{
			"bucket":                     "gs://some_folder-2",
			tenant.ProjectSchedulerHost:  "host",
			tenant.ProjectStoragePathKey: "gs://location",
		}, map[string]string{})
	namespace, _ := tenant.NewNamespace("test-ns", proj.Name(),
		map[string]string{
			"bucket": "gs://ns_bucket",
		}, map[string]string{})
	otherNamespace, _ := tenant.NewNamespace("other-ns", proj.Name(),
		map[string]string{
			"bucket": "gs://ns_bucket",
		}, map[string]string{})

	dbSetup := func() *pgxpool.Pool {
		pool := setup.TestPool()
		setup.TruncateTablesWith(pool)

		projRepo := postgres.NewProjectRepository(pool)
		assert.Nil(t, projRepo.Save(ctx, proj))

		namespaceRepo := postgres.NewNamespaceRepository(pool)
		assert.Nil(t, namespaceRepo.Save(ctx, namespace))
		assert.Nil(t, namespaceRepo.Save(ctx, otherNamespace))

		return pool
	}

	t.Run("Save", func(t *testing.T) {
		t.Run("inserts the secret without namespace set", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name", "abcd", proj.Name(), "")
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)

			secret, err := repo.Get(ctx, proj.Name(), "", validSecret.Name())
			assert.Nil(t, err)
			assert.Equal(t, validSecret.Name(), secret.Name())
			assert.Equal(t, validSecret.EncodedValue(), secret.EncodedValue())

			nsName := secret.NamespaceName()
			assert.Equal(t, "", nsName)
		})
		t.Run("inserts the secret with namespace set", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name", "abcd",
				proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)

			secret, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.Nil(t, err)
			assert.Equal(t, validSecret.Name(), secret.Name())
			assert.Equal(t, validSecret.EncodedValue(), secret.EncodedValue())

			assert.Equal(t, namespace.Name().String(), secret.NamespaceName())
		})
		t.Run("returns error when same secret is inserted twice", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)

			secret, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.Nil(t, err)
			assert.Equal(t, validSecret.Name(), secret.Name())

			err = repo.Save(ctx, validSecret)
			assert.NotNil(t, err)
		})
	})
	t.Run("Update", func(t *testing.T) {
		t.Run("updates an already existing resource", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)
			validSecret2, err := tenant.NewSecret("secret_name_2",
				"efgh", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)
			err = repo.Save(ctx, validSecret2)
			assert.Nil(t, err)

			secret, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.Nil(t, err)
			assert.Equal(t, validSecret.Name(), secret.Name())
			assert.Equal(t, validSecret.EncodedValue(), secret.EncodedValue())

			updatedSecret, err := tenant.NewSecret("secret_name",
				"efgh", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			err = repo.Update(ctx, updatedSecret)
			assert.Nil(t, err)

			updated, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.Nil(t, err)
			assert.Equal(t, updatedSecret.Name(), updated.Name())
			assert.Equal(t, updatedSecret.EncodedValue(), updated.EncodedValue())

			unUpdated, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret2.Name())
			assert.Nil(t, err)
			assert.Equal(t, unUpdated.Name(), validSecret2.Name())
			assert.Equal(t, unUpdated.EncodedValue(), validSecret2.EncodedValue())
		})
		t.Run("returns error when secret does not exist", func(t *testing.T) {
			db := dbSetup()

			repo := postgres.NewSecretRepository(db)

			updatedSecret, err := tenant.NewSecret("secret_name",
				"efgh", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			err = repo.Update(ctx, updatedSecret)
			assert.NotNil(t, err)
			assert.EqualError(t, err, "not found for entity secret: unable to update, secret not found for SECRET_NAME")
		})
	})
	t.Run("Get", func(t *testing.T) {
		t.Run("returns error when record is not present", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			_, err = repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.NotNil(t, err)
			assert.EqualError(t, err, "not found for entity secret: no record for SECRET_NAME")
		})
		t.Run("returns the secret when present", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)

			secret, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.Nil(t, err)
			assert.Equal(t, validSecret.Name(), secret.Name())
			assert.Equal(t, validSecret.EncodedValue(), secret.EncodedValue())

			assert.Equal(t, proj.Name().String(), secret.ProjectName().String())
			assert.Equal(t, namespace.Name().String(), secret.NamespaceName())
		})
		t.Run("should get all the secrets info for a project", func(t *testing.T) {
			db := dbSetup()

			repo := postgres.NewSecretRepository(db)

			secret1, err := tenant.NewSecret("secret_name1", "abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret1)
			assert.Nil(t, err)

			secret2, err := tenant.NewSecret("secret_name3", "abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret2)
			assert.Nil(t, err)

			sn, err := tenant.SecretNameFrom("secret_name3")
			assert.Nil(t, err)
			secret, err := repo.Get(ctx, proj.Name(), namespace.Name().String(), sn)
			assert.NoError(t, err)
			assert.NotNil(t, secret)
			assert.Equal(t, "SECRET_NAME3", secret.Name().String())
		})
	})
	t.Run("GetAll", func(t *testing.T) {
		t.Run("returns all the secrets for a project", func(t *testing.T) {
			db := dbSetup()

			repo := postgres.NewSecretRepository(db)

			secret1, err := tenant.NewSecret("secret_name1",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret1)
			assert.Nil(t, err)

			secret2, err := tenant.NewSecret("secret_name2",
				"abcd", proj.Name(), otherNamespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret2)
			assert.Nil(t, err)

			secret3, err := tenant.NewSecret("secret_name3",
				"abcd", proj.Name(), "")
			assert.Nil(t, err)
			err = repo.Save(ctx, secret3)
			assert.Nil(t, err)

			secrets, err := repo.GetAll(ctx, proj.Name(), "")
			assert.Nil(t, err)

			assert.Equal(t, 3, len(secrets))

			assert.Equal(t, secret1.Name(), secrets[0].Name())
			assert.Equal(t, secret2.Name(), secrets[1].Name())
			assert.Equal(t, secret3.Name(), secrets[2].Name())
		})
		t.Run("returns secrets for current namespace and ones without namespace in project", func(t *testing.T) {
			db := dbSetup()

			repo := postgres.NewSecretRepository(db)

			secret1, err := tenant.NewSecret("secret_name1",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret1)
			assert.Nil(t, err)

			secret2, err := tenant.NewSecret("secret_name2",
				"abcd", proj.Name(), otherNamespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret2)
			assert.Nil(t, err)

			secret3, err := tenant.NewSecret("secret_name3",
				"abcd", proj.Name(), "")
			assert.Nil(t, err)
			err = repo.Save(ctx, secret3)
			assert.Nil(t, err)

			secrets, err := repo.GetAll(ctx, proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			assert.Equal(t, 2, len(secrets))

			assert.Equal(t, secret1.Name(), secrets[0].Name())
			assert.Equal(t, secret3.Name(), secrets[1].Name())
		})
	})
	t.Run("Delete", func(t *testing.T) {
		t.Run("deletes the secret for namespace", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)

			err = repo.Delete(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.Nil(t, err)

			_, err = repo.Get(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.NotNil(t, err)
		})
		t.Run("deletes the secret for project", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), "")
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Save(ctx, validSecret)
			assert.Nil(t, err)

			err = repo.Delete(ctx, proj.Name(), "", validSecret.Name())
			assert.Nil(t, err)

			_, err = repo.Get(ctx, proj.Name(), "", validSecret.Name())
			assert.NotNil(t, err)
		})
		t.Run("returns error when non existing is deleted", func(t *testing.T) {
			db := dbSetup()

			validSecret, err := tenant.NewSecret("secret_name",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)

			repo := postgres.NewSecretRepository(db)

			err = repo.Delete(ctx, proj.Name(), namespace.Name().String(), validSecret.Name())
			assert.NotNil(t, err)
			assert.EqualError(t, err, "not found for entity secret: secret to delete not found SECRET_NAME")
		})
	})
	t.Run("GetSecretsInfo", func(t *testing.T) {
		t.Run("should get all the secrets info for a project", func(t *testing.T) {
			db := dbSetup()

			repo := postgres.NewSecretRepository(db)

			secret1, err := tenant.NewSecret("secret_name1",
				"abcd", proj.Name(), namespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret1)
			assert.Nil(t, err)

			secret2, err := tenant.NewSecret("secret_name2",
				"abcd", proj.Name(), otherNamespace.Name().String())
			assert.Nil(t, err)
			err = repo.Save(ctx, secret2)
			assert.Nil(t, err)

			secret3, err := tenant.NewSecret("secret_name3",
				"abcd", proj.Name(), "")
			assert.Nil(t, err)
			err = repo.Save(ctx, secret3)
			assert.Nil(t, err)

			projScopeSecrets, err := repo.GetSecretsInfo(ctx, proj.Name())
			assert.Nil(t, err)
			assert.Equal(t, 3, len(projScopeSecrets))

			// returns only at project scope, ignores the namespace
			secretsInfo, err := repo.GetSecretsInfo(ctx, proj.Name())
			assert.Nil(t, err)
			assert.Equal(t, 3, len(secretsInfo))

			info1 := secretsInfo[0]
			assert.Equal(t, secret1.Name().String(), info1.Name)
			assert.Equal(t, namespace.Name().String(), info1.Namespace)
			assert.NotNil(t, info1.UpdatedAt)
			assert.NotEmpty(t, info1.Digest)

			info2 := secretsInfo[1]
			assert.Equal(t, secret2.Name().String(), info2.Name)
			assert.Equal(t, otherNamespace.Name().String(), info2.Namespace)
			assert.NotNil(t, info2.UpdatedAt)
			assert.NotEmpty(t, info2.Digest)

			info3 := secretsInfo[2]
			assert.Equal(t, secret3.Name().String(), info3.Name)
			assert.Equal(t, "", info3.Namespace)
			assert.NotNil(t, info3.UpdatedAt)
			assert.NotEmpty(t, info3.Digest)
		})
	})
}
