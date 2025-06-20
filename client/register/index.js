import * as api from "../api.js";

const usernameField = document.getElementById("usernameField");
const passwordField = document.getElementById("passwordField");

async function onSubmitButtonClick() {
  const usernameFieldValue = usernameField.value.trim();
  const passwordFieldValue = passwordField.value;

  if (!usernameFieldValue) {
    alert("⚠️ Please enter your username");
    return;
  }

  if (!passwordFieldValue) {
    alert("⚠️ Please enter your password");
    return;
  }

  await api.register(usernameFieldValue, passwordFieldValue);

  document.getElementById("message").innerHTML =
    '✅ Successfully registered an account!<br/><a href="/login">Click here</a> to log in.';
}

await (async () => {
  const needsLogin = await api.needsLogin();
  if (!needsLogin) {
    window.location.replace("/");
  }

  const submitButton = document.getElementById("submitButton");
  submitButton.onclick = async (e) => await onSubmitButtonClick(e);
})();
