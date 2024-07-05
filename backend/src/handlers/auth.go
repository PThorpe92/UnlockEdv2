package handlers

import (
	"UnlockEdv2/src/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
)

type contextKey string

const ClaimsKey contextKey = "claims"

type Claims struct {
	UserID        uint            `json:"user_id"`
	PasswordReset bool            `json:"password_reset"`
	Role          models.UserRole `json:"role"`
	FacilityID    uint            `json:"facility_id"`
	jwt.RegisteredClaims
}

func (srv *Server) registerAuthRoutes() {
	srv.Mux.Handle("POST /api/reset-password", srv.applyMiddleware(http.HandlerFunc(srv.handleResetPassword)))
	/* only use auth middleware, user activity bloats the database + results */
	srv.Mux.Handle("GET /api/auth", srv.AuthMiddleware(http.HandlerFunc(srv.handleCheckAuth)))
	srv.Mux.Handle("PUT /api/admin/facility-context/{id}", srv.ApplyAdminMiddleware(http.HandlerFunc(srv.handleChangeAdminFacility)))
}

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("unlocked_token")
		if err != nil {
			log.Error("No token found " + err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("APP_KEY")), nil
		})
		if err != nil {
			log.Println("Invalid token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			if claims.PasswordReset && !isAuthRoute(r) {
				http.Redirect(w, r.WithContext(ctx), "/reset-password", http.StatusOK)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			log.Println("Invalid claims")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}

func (srv *Server) handleChangeAdminFacility(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "couldn't parse id from path", http.StatusBadRequest)
		return
	}
	claims := r.Context().Value(ClaimsKey).(*Claims)
	claims.FacilityID = uint(id)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(os.Getenv("APP_KEY")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "unlocked_token",
		Value:    signedToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
}

func isAuthRoute(r *http.Request) bool {
	paths := []string{"/api/login", "/api/logout", "/api/reset-password", "/api/auth", "/api/consent/accept"}
	return slices.Contains(paths, r.URL.Path)
}

func (srv *Server) UserIsAdmin(r *http.Request) bool {
	claims := r.Context().Value(ClaimsKey).(*Claims)
	return claims.Role == models.Admin
}

func (srv *Server) canViewUserData(r *http.Request) bool {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return false
	}
	claims := r.Context().Value(ClaimsKey).(*Claims)
	return claims.Role == models.Admin || claims.UserID == uint(id)
}

func (srv *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(ClaimsKey).(*Claims)
		if !ok {
			http.Error(w, "Unauthorized - no claims", http.StatusUnauthorized)
			return
		}
		if claims.Role != models.Admin {
			http.Error(w, "Unauthorized - not admin", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}

type OrySession struct {
	Active    bool
	TokenHash string
	Expires   time.Time
}

func hashCookieValue(cookieValue string) string {
	hash := sha256.New()
	hash.Write([]byte(cookieValue))
	return hex.EncodeToString(hash.Sum(nil))
}

// cache ory sessions in memory to avoid unnecessary calls
var orySessions = make(map[uint]OrySession)

// Auth endpoint that is called from the client before each <AuthenticatedLayout /> is rendered
func (srv *Server) handleCheckAuth(w http.ResponseWriter, r *http.Request) {
	fields := log.Fields{"handler": "handleCheckAuth"}
	claims := r.Context().Value(ClaimsKey).(*Claims)
	user, err := srv.Db.GetUserByID(claims.UserID)
	if err != nil {
		log.Error("Error getting user by ID")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	fields["user.username"] = user.Username
	fields["facility_id"] = user.FacilityID
	if user.Role != models.Admin && user.FacilityID != claims.FacilityID {
		// user isn't an admin, and has alternate facility_id in the JWT claims
		fields["claims.facility_id"] = claims.FacilityID
		log.WithFields(fields).Error("user viewing context for different facility. this should never happen")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	oryCookie, err := r.Cookie("ory_kratos_session")
	if err == nil {
		cookieHash := hashCookieValue(oryCookie.Value)
		if orySession, ok := orySessions[user.ID]; ok {
			// check the cached session hash + expiration
			if orySession.Active && orySession.TokenHash == cookieHash && orySession.Expires.After(time.Now()) {
				srv.WriteResponse(w, http.StatusOK, user)
				log.WithFields(fields).Info("checked cached session active for user")
				return
			}
			log.WithFields(fields).Info("session was cached but invalidated")
		}
		if err := srv.validateOrySession(oryCookie, user.ID); err != nil {
			log.WithFields(fields).Errorln("invalid ory session found")
			srv.ErrorResponse(w, http.StatusUnauthorized, "ory session was not valid, please login again")
			return
		}
		srv.WriteResponse(w, http.StatusOK, user)
		return
	}
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// we pull the ory session cookie, and send request to kratos to validate the user session
func (srv *Server) validateOrySession(cookie *http.Cookie, userID uint) error {
	fields := log.Fields{"handler": "validateOrySession", "user_id": userID}
	request, err := http.NewRequest("GET", os.Getenv("KRATOS_PUBLIC_URL")+"/sessions/whoami", nil)
	if err != nil {
		log.WithFields(fields).Error("Error creating request to ory: ", err)
		return err
	}
	// for some reason you have to send the entire cookie, instead of cookie.Value
	request.Header.Add("Cookie", cookie.String())
	response, err := srv.Client.Do(request)
	if err != nil {
		log.WithFields(fields).Error("Error sending equest to ory: ", err)
		return err
	}
	if response.StatusCode == 200 {
		oryResp := map[string]interface{}{}
		if err := json.NewDecoder(response.Body).Decode(&oryResp); err != nil {
			log.WithFields(fields).Errorln("error decoding body from ory response")
			return err
		}
		defer response.Body.Close()
		active, ok := oryResp["active"].(bool)
		if !ok {
			log.WithFields(fields).Errorln("error decoding active session from ory response")
			return err
		}
		if active {
			expiresAt, ok := oryResp["expires_at"].(string)
			if !ok {
				log.WithFields(fields).Errorln("error expires_at from ory response")
				return err
			}
			expires, err := time.Parse(time.RFC3339, expiresAt)
			if err != nil {
				log.WithFields(fields).Errorln("error parsing expires_at time from ory response")
				return err
			}
			if expires.After(time.Now()) {
				log.WithFields(fields).Info("Got active  session from ory")
				// hash the ory token for easy comparison/validation, so we cache
				hashed := hashCookieValue(cookie.Value)
				orySessions[userID] = OrySession{Active: true, TokenHash: hashed, Expires: expires}
				return nil
			}
		}
	}
	delete(orySessions, userID)
	return errors.New("invalid ory session")
}

type ResetPasswordRequest struct {
	Password string `json:"password"`
	Confirm  string `json:"confirm"`
}

func (srv *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	log.Info("Handling password reset request", r.URL.Path)
	claims := r.Context().Value(ClaimsKey).(*Claims)
	form := ResetPasswordRequest{}
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		log.Error("Parsing form failed, using JSON" + err.Error())
	}
	password := form.Password
	confirm := form.Confirm
	defer r.Body.Close()
	if password != confirm {
		http.Error(w, "Passwords do not match", http.StatusBadRequest)
		return
	}
	if !validatePassword(password) {
		http.Error(w, "Password must be at least 8 characters long and contain a number", http.StatusBadRequest)
		return
	}
	if err := srv.Db.ResetUserPassword(claims.UserID, password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error("Error Resetting users password: " + err.Error())
		return
	}
	claims.PasswordReset = false
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(os.Getenv("APP_KEY")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "unlocked_token",
		Value:    signedToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
	})
	if !srv.isTesting(r) {
		user, err := srv.Db.GetUserByID(claims.UserID)
		if err != nil {
			log.Fatal("user from claims not found, this should never happen")
			return
		}
		if err := srv.handleUpdatePasswordKratos(user, password); err != nil {
			log.Errorln("Error updating password in kratos: ", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	srv.WriteResponse(w, http.StatusOK, "Password reset successfully")
}

func validatePassword(pass string) bool {
	if len(pass) < 8 || !strings.ContainsAny(pass, "12345678910") {
		return false
	}
	return true
}
