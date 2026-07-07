package install

import "testing"

func TestNormalizeOptionsUsesInstallDefaults(t *testing.T) {
	opts, err := NormalizeOptions(Options{
		DatabaseURL: "postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable",
	})
	if err != nil {
		t.Fatalf("NormalizeOptions returned error: %v", err)
	}

	if opts.AdminEmail != "admin@example.com" {
		t.Fatalf("AdminEmail = %q, want admin@example.com", opts.AdminEmail)
	}
	if opts.AdminNickname != "admin" {
		t.Fatalf("AdminNickname = %q, want admin", opts.AdminNickname)
	}
	if opts.AdminPassword != "123456" {
		t.Fatalf("AdminPassword = %q, want 123456", opts.AdminPassword)
	}
	if opts.MigrationsDir != "migrations" {
		t.Fatalf("MigrationsDir = %q, want migrations", opts.MigrationsDir)
	}
	if !opts.ResetAdminPassword {
		t.Fatal("ResetAdminPassword = false, want true")
	}
}

func TestNormalizeOptionsTrimsAndValidatesAdminEmail(t *testing.T) {
	opts, err := NormalizeOptions(Options{
		DatabaseURL:    "postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable",
		AdminEmail:     "  ADMIN@Example.COM  ",
		AdminPassword:  "secret",
		AdminNickname:  " Root ",
		MigrationsDir:  " ./migrations ",
		SkipAdminReset: true,
	})
	if err != nil {
		t.Fatalf("NormalizeOptions returned error: %v", err)
	}

	if opts.AdminEmail != "admin@example.com" {
		t.Fatalf("AdminEmail = %q, want admin@example.com", opts.AdminEmail)
	}
	if opts.AdminNickname != "Root" {
		t.Fatalf("AdminNickname = %q, want Root", opts.AdminNickname)
	}
	if opts.MigrationsDir != "./migrations" {
		t.Fatalf("MigrationsDir = %q, want ./migrations", opts.MigrationsDir)
	}
	if opts.ResetAdminPassword {
		t.Fatal("ResetAdminPassword = true, want false")
	}
}

func TestNormalizeOptionsRejectsInvalidInputs(t *testing.T) {
	cases := []Options{
		{},
		{DatabaseURL: "postgres://postgres:123456@127.0.0.1:5432/image-ai?sslmode=disable", AdminEmail: "admin"},
	}

	for _, tc := range cases {
		if _, err := NormalizeOptions(tc); err == nil {
			t.Fatalf("NormalizeOptions(%+v) returned nil error", tc)
		}
	}
}
