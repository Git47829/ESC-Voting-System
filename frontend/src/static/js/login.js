const tokenInput = document.getElementById("token");
const toggleBtn  = document.getElementById("toggle-visibility");
const eyeIcon    = document.getElementById("eye-icon");
const eyeOffIcon = document.getElementById("eye-off-icon");

toggleBtn.addEventListener("click", function () {
    const isPassword = tokenInput.type === "password";
    tokenInput.type  = isPassword ? "text" : "password";
    eyeIcon.classList.toggle("hidden", isPassword);
    eyeOffIcon.classList.toggle("hidden", !isPassword);
});

tokenInput.focus();
