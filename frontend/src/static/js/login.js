const passwordInput = document.getElementById("password");
const toggleBtn = document.getElementById("toggle-visibility");
const eyeIcon = document.getElementById("eye-icon");
const eyeOffIcon = document.getElementById("eye-off-icon");

if (toggleBtn && passwordInput) {
    toggleBtn.addEventListener("click", function () {
        const isPassword = passwordInput.type === "password";
        passwordInput.type = isPassword ? "text" : "password";
        eyeIcon.classList.toggle("hidden", isPassword);
        eyeOffIcon.classList.toggle("hidden", !isPassword);
    });
}

// Auto-focus: code input on step 2, email on step 1
const codeInput = document.getElementById("code");
const emailInput = document.getElementById("email");
if (codeInput) {
    codeInput.focus();
} else if (emailInput) {
    emailInput.focus();
}
