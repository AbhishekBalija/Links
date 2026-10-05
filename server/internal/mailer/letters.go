package mailer

// Every email LINKS sends, written once: the Resend and SMTP mailers only
// deliver them.

// email is one written message, ready to deliver.
type email struct {
	To      string
	Subject string
	HTML    string
}

func signInCodeEmail(to, code string) email {
	return email{To: to, Subject: "Your LINKS sign-in code: " + code, HTML: signInCodeHTML(code)}
}

func staffAddedEmail(to string, letter StaffAdded, signInURL string) email {
	return email{To: to, Subject: "You were added to LINKS", HTML: staffAddedHTML(to, letter, signInURL)}
}

func accessDecisionEmail(to string, letter AccessDecision, signInURL string) email {
	subject := "You're in: your LINKS request was approved"
	if !letter.Approved {
		subject = "Your LINKS request wasn't approved"
	}
	return email{To: to, Subject: subject, HTML: accessDecisionHTML(letter, signInURL)}
}

func eventNoticeEmails(to []Recipient, letter EventNotice, signInURL string) []email {
	subject := "Changed: " + letter.Title
	if letter.Cancelled {
		subject = "Cancelled: " + letter.Title + ", " + letter.When
	}
	emails := make([]email, 0, len(to))
	for _, recipient := range to {
		emails = append(emails, email{To: recipient.Email, Subject: subject, HTML: eventNoticeHTML(recipient, letter, signInURL)})
	}
	return emails
}

func applicationUpdateEmail(to string, letter ApplicationUpdate, signInURL string) email {
	subject := "Update on " + letter.Title + " at " + letter.Company
	switch letter.Status {
	case "shortlisted":
		subject = "Shortlisted: " + letter.Title + " at " + letter.Company
	case "selected":
		subject = "Selected: " + letter.Title + " at " + letter.Company
	}
	return email{To: to, Subject: subject, HTML: applicationUpdateHTML(letter, signInURL)}
}
