var navLinks = document.querySelectorAll("nav a");
for (let i = 0; i < navLinks.length; i++) {
	let link = navLinks[i];
	if (link.getAttribute('href') === window.location.pathname) {
		link.classList.add("live");
		break;
	}
}