function removeCommentsFromJson(json) {
  let result = "";
  let inString = false;
  let escaped = false;

  for (let i = 0; i < json.length; ) {
    const c = json[i];

    if (c === '"' && !escaped) {
      inString = !inString;
    }

    if (c === "\\" && inString) {
      escaped = !escaped;
    } else {
      escaped = false;
    }

    if (!inString && c === "/" && i + 1 < json.length) {
      const next = json[i + 1];

      if (next === "/") {
        i += 2;
        while (i < json.length && json[i] !== "\n" && json[i] !== "\r") {
          i++;
        }
        continue;
      }

      if (next === "*") {
        i += 2;
        while (i + 1 < json.length) {
          if (json[i] === "*" && json[i + 1] === "/") {
            i += 2;
            break;
          }
          i++;
        }
        if (i + 1 >= json.length) {
          throw new Error("Unclosed comment");
        }
        continue;
      }
    }

    result += c;
    i++;
  }

  return result;
}

export async function loadJson(jsonPath) {
  return fetch(jsonPath)
    .then((res) => res.text())
    .then((jsonText) => JSON.parse(removeCommentsFromJson(jsonText)));
}
