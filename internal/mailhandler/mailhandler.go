package mailhandler

import (
	"encoding/json"
	"github.com/OliverSchlueter/goutils/problems"
	"github.com/OliverSchlueter/mail-server/internal/mails"
	"github.com/OliverSchlueter/mail-server/internal/smtp"
	"github.com/OliverSchlueter/mail-server/internal/users"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	mailStore mails.Store
	userStore users.Store
}

func New(mailStore mails.Store, userStore users.Store) *Handler {
	return &Handler{
		mailStore: mailStore,
		userStore: userStore,
	}
}

func (h *Handler) Register(prefix string, mux *http.ServeMux) {
	mux.HandleFunc(prefix+"/mailboxes/{user_id}/", h.handleMailboxes)
	mux.HandleFunc(prefix+"/mailboxes/{user_id}/{mailbox}", h.handleMailbox)
	mux.HandleFunc(prefix+"/mailboxes/{user_id}/{mailbox}/mails", h.handleMails)
	mux.HandleFunc(prefix+"/mailboxes/{user_id}/{mailbox}/mails/{mail}", h.handleMail)
}

func (h *Handler) handleMailboxes(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("user_id")

	switch r.Method {
	case http.MethodGet:
		h.getMailboxes(w, r, userId)
	default:
		problems.MethodNotAllowed(r.Method, []string{http.MethodGet}).WriteToHTTP(w)
	}
}

func (h *Handler) getMailboxes(w http.ResponseWriter, r *http.Request, userId string) {
	mailboxes, err := h.mailStore.GetMailboxes(userId)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	data, err := json.Marshal(mailboxes)
	if err != nil {
		problems.InternalServerError("Error marshalling mailboxes").WriteToHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) handleMailbox(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("user_id")
	mailboxName := r.PathValue("mailbox")

	switch r.Method {
	case http.MethodGet:
		h.getMailbox(w, r, userId, mailboxName)
	default:
		problems.MethodNotAllowed(r.Method, []string{http.MethodGet}).WriteToHTTP(w)
	}
}

func (h *Handler) getMailbox(w http.ResponseWriter, r *http.Request, userId string, mailboxName string) {
	mailbox, err := h.mailStore.GetMailboxByName(userId, mailboxName)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	data, err := json.Marshal(mailbox)
	if err != nil {
		problems.InternalServerError("Error marshalling mailbox").WriteToHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) handleMails(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("user_id")
	mailboxName := r.PathValue("mailbox")

	switch r.Method {
	case http.MethodGet:
		h.getMails(w, r, userId, mailboxName)
	case http.MethodPost:
		h.createMail(w, r, userId, mailboxName)
	default:
		problems.MethodNotAllowed(r.Method, []string{http.MethodGet, http.MethodPost}).WriteToHTTP(w)
	}
}

func (h *Handler) getMails(w http.ResponseWriter, r *http.Request, userId string, mailboxName string) {
	mailbox, err := h.mailStore.GetMailboxByName(userId, mailboxName)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	m, err := h.mailStore.GetMails(userId, mailbox.UID)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	data, err := json.Marshal(m)
	if err != nil {
		problems.InternalServerError("Error marshalling mails").WriteToHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) createMail(w http.ResponseWriter, r *http.Request, userId string, mailboxName string) {
	var req CreateMailReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		problems.CouldNotDecodeBody().WriteToHTTP(w)
		return
	}

	user, err := h.userStore.GetByName(userId)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	msgID := time.Now().Format("20060102150405") + "@" + user.PrimaryEmail
	date := time.Now().Format(time.RFC1123Z)

	dataBuffer := []string{
		"Date: " + date,
		"MIME-Version: 1.0",
		"Message-ID: <" + msgID + ">",
		"Subject: " + req.Subject,
		"From: <" + user.PrimaryEmail + ">",
		"To: <" + req.To[0] + ">", // Assuming single recipient for simplicity
		"Content-Transfer-Encoding: quoted-printable",
		"Content-Type: text/html; charset=UTF-8",
		"",
		req.Body,
	}

	smtpMail := smtp.Mail{
		Outgoing:    true,
		From:        user.PrimaryEmail,
		To:          req.To,
		DataBuffer:  dataBuffer,
		ReadingData: false,
	}

	recepientsCount, err := smtp.SendMail(smtpMail)
	if err != nil {
		problems.InternalServerError("Failed to send mail: " + err.Error()).WriteToHTTP(w)
		return
	}

	if recepientsCount != len(req.To) {
		problems.InternalServerError("Not all recipients were sent the mail").WriteToHTTP(w)
		return
	}

	mailbox, err := h.mailStore.GetMailboxByName(userId, mailboxName)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	mailsMail := mails.Mail{
		UID:        mails.RandomUID(),
		MailboxUID: mailbox.UID,
		Flags:      []string{},
		Date:       time.Now(),
		Size:       len(req.Body),
		Headers: map[string]string{
			"From":       user.PrimaryEmail,
			"To":         req.To[0], // Assuming single recipient for simplicity
			"Subject":    req.Subject,
			"Date":       date,
			"Message-ID": msgID,
		},
		Body: req.Body,
	}
	if err := h.mailStore.CreateMail(userId, mailbox.UID, mailsMail); err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) handleMail(w http.ResponseWriter, r *http.Request) {
	userId := r.PathValue("user_id")
	mailboxName := r.PathValue("mailbox")
	mailUID := r.PathValue("mail")

	switch r.Method {
	case http.MethodGet:
		h.getMail(w, r, userId, mailboxName, mailUID)
	default:
		problems.MethodNotAllowed(r.Method, []string{http.MethodGet}).WriteToHTTP(w)
	}
}

func (h *Handler) getMail(w http.ResponseWriter, r *http.Request, userId string, mailboxName string, mailUID string) {
	mailbox, err := h.mailStore.GetMailboxByName(userId, mailboxName)
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	uid, err := strconv.ParseUint(mailUID, 10, 32)
	if err != nil {
		problems.ValidationError("Mail UID", "Invalid mail UID").WriteToHTTP(w)
		return
	}

	mail, err := h.mailStore.GetMailByUID(userId, mailbox.UID, uint32(uid))
	if err != nil {
		problems.InternalServerError(err.Error()).WriteToHTTP(w)
		return
	}

	data, err := json.Marshal(mail)
	if err != nil {
		problems.InternalServerError("Error marshalling mail").WriteToHTTP(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
