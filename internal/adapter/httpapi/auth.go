package httpapi

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

func requireUser(w http.ResponseWriter, r *http.Request, db *DB, requestID string) (User, bool) {
	user, ok := currentUser(db, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Chua dang nhap.", nil, requestID)
	}
	return user, ok
}

func currentUser(db *DB, r *http.Request) (User, bool) {
	token := bearerToken(r)
	if token == "" {
		return User{}, false
	}
	tokenHash := sha256Hex(token)
	for _, session := range db.Sessions {
		if session.TokenHash == tokenHash {
			expiresAt, err := parseSessionTime(session.ExpiresAt)
			if err != nil || expiresAt.Before(time.Now()) {
				return User{}, false
			}
			for _, user := range db.Users {
				if user.ID == session.UserID {
					return user, true
				}
			}
		}
	}
	return User{}, false
}

func bearerToken(r *http.Request) string {
	parts := strings.SplitN(r.Header.Get("authorization"), " ", 2)
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

func createSession(db *DB, userID string) string {
	token, session := buildSession(userID)
	db.Sessions = append(db.Sessions, session)
	return token
}

func buildSession(userID string) (string, Session) {
	token := randomHex(32)
	now := time.Now().UTC()
	return token, Session{TokenHash: sha256Hex(token), UserID: userID, CreatedAt: now.Format(time.RFC3339Nano), ExpiresAt: now.Add(30 * 24 * time.Hour).Format(time.RFC3339Nano)}
}

func removeSession(db *DB, token string) bool {
	tokenHash := sha256Hex(token)
	next := db.Sessions[:0]
	changed := false
	for _, session := range db.Sessions {
		if session.TokenHash == tokenHash {
			changed = true
			continue
		}
		next = append(next, session)
	}
	db.Sessions = next
	return changed
}

func pruneExpiredSessions(db *DB) bool {
	now := time.Now()
	next := db.Sessions[:0]
	changed := false
	for _, session := range db.Sessions {
		expiresAt, err := parseSessionTime(session.ExpiresAt)
		if err != nil || expiresAt.Before(now) {
			changed = true
			continue
		}
		next = append(next, session)
	}
	db.Sessions = next
	return changed
}

func parseSessionTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
	}
	return parsed, err
}

func hashPassword(password string, salt string) (string, string) {
	if salt == "" {
		salt = randomHex(16)
	}
	hashValue, err := pbkdf2.Key[hash.Hash](sha512.New, password, []byte(salt), 210000, 64)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hashValue), salt
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil && !strings.Contains(email, " ")
}

func sanitizeUser(user User) map[string]any {
	return map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "createdAt": user.CreatedAt}
}

func sha256Hex(value string) string {
	hashValue := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hashValue[:])
}
