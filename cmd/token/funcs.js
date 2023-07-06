function copy2clipboard() {{
    var elem = document.getElementById("token");
    elem.style.display = "block";
    elem.select();
    document.execCommand("copy");
    elem.style.display = "none";
}}