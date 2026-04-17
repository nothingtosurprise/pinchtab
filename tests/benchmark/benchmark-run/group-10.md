# Group 10: Nested Interactions & Modal Dialogs

### 10.1 Open and interact with modal on dashboard
Navigate to `http://fixtures/dashboard.html`. Find and click the Settings button (selector: `#settings-btn`) to open the modal dialog.

**Verify**: Modal appeared — snapshot contains "Dashboard Settings".

### 10.2 Modify settings and close modal
In the modal, select "Dark" from the theme dropdown (`#theme-select`), then click the Save button (`#modal-save`). After the modal closes, check the page content.

**Verify**: Page contains `THEME_DARK_APPLIED`.

---

