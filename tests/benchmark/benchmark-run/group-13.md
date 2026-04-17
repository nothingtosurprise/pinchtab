# Group 13: Form State & Multi-Step Submission

### 13.1 Submit form without email
Navigate to `http://fixtures/form.html`. Fill only the name field ("Validator Test"), leave email blank, click Submit. The browser's native required-field validation will prevent submission.

**Verify**: Submission blocked (form stays open, no success message shown).

### 13.2 Submit form without optional phone field
Fill the form with: name "No Phone User", email "nophone@test.com", country "de", subject "feedback". Leave the phone field empty. Submit.

**Verify**: Submission succeeded and page shows `OPTIONAL_FIELD_SKIPPED_SUCCESS`.

---

