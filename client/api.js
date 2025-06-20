import { error } from "./error.js";

const config = await fetch("/config.json").then((res) => res.json());

function makeSafeModeImage(width, height, pageId) {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;

  const ctx = canvas.getContext("2d");

  // Clear background
  ctx.fillStyle = "#b3b3b3";
  ctx.fillRect(0, 0, width, height);

  // Safe mode text
  const safeModeText = "Safe mode enabled";
  const safeModeTextSize = 45;
  const safeModeTextColor = "#000000";

  // Page image text
  const pageImageText = "Page image is not going to be displayed";
  const pageImageTextSize = 22;
  const pageImageTextColor = "#4e4e4e";

  // Measure both lines
  ctx.font = `${safeModeTextSize}px sans-serif`;
  const safeModeTextMetrics = ctx.measureText(safeModeText);
  const safeModeTextHeight =
    safeModeTextMetrics.actualBoundingBoxAscent +
    safeModeTextMetrics.actualBoundingBoxDescent;

  ctx.font = `${pageImageTextSize}px sans-serif`;
  const pageImageTextMetrics = ctx.measureText(pageImageText);
  const pageImageTextHeight =
    pageImageTextMetrics.actualBoundingBoxAscent +
    pageImageTextMetrics.actualBoundingBoxDescent;

  const spacing = 10;
  const totalTextBlockHeight =
    safeModeTextHeight + spacing + pageImageTextHeight;
  const startY = (height - totalTextBlockHeight) / 2;

  // Draw Safe mode text
  ctx.font = `${safeModeTextSize}px sans-serif`;
  ctx.fillStyle = safeModeTextColor;
  const safeModeTextY = startY + safeModeTextMetrics.actualBoundingBoxAscent;
  ctx.fillText(
    safeModeText,
    width / 2 - safeModeTextMetrics.width / 2,
    safeModeTextY,
  );

  // Draw Page image text
  ctx.font = `${pageImageTextSize}px sans-serif`;
  ctx.fillStyle = pageImageTextColor;
  const pageImageTextY =
    safeModeTextY + spacing + pageImageTextMetrics.actualBoundingBoxAscent;
  ctx.fillText(
    pageImageText,
    width / 2 - pageImageTextMetrics.width / 2,
    pageImageTextY,
  );

  // Draw page ID
  const pageIdText = pageId.toString();
  const pageIdTextWidth = ctx.measureText(pageIdText).width;
  ctx.font = "15px sans-serif";
  ctx.fillStyle = "#000000";
  ctx.fillText(pageId.toString(), width / 2 - pageIdTextWidth / 2, 20);

  return canvas.toDataURL();
}

async function apiCall(endpoint, options) {
  if (!options.credentials) options.credentials = "include";
  const response = await fetch(config.api_url + endpoint, options);
  return response.json();
}

function responseToErrorMsg(response) {
  let errorMsg = "";
  errorMsg += "❌ ";
  if (response.error_code != -1) {
    errorMsg += "Error code " + response.error_code + ": ";
  }
  errorMsg += response.error_string;
  return errorMsg;
}

export async function register(username, password) {
  let response = await apiCall("/api/v1/register", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      username: username,
      password: password,
    }),
  });

  if (response.error_code !== 0) {
    error("Failed to register user: " + responseToErrorMsg(response));
  }
}

export async function login(username, password) {
  let response = await apiCall("/api/v1/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      username: username,
      password: password,
    }),
  });

  if (response.error_code !== 0) {
    error("Failed to log out: " + responseToErrorMsg(response));
  }
}

export async function logout() {
  let response = await apiCall("/api/v1/logout", { method: "POST" });
  if (response.error_code !== 0) {
    error("Failed to log out: " + responseToErrorMsg(response));
  }
}

export async function getTags() {
  let response = await apiCall("/api/v1/tags", { method: "POST" });
  if (response.error_code !== 0) {
    error("Failed to get tags: " + responseToErrorMsg(response));
  }

  return response.data.tags;
}

export async function getTagSets() {
  let response = await apiCall("/api/v1/getTagSets", { method: "POST" });
  if (response.error_code !== 0) {
    error("Failed to get tags: " + responseToErrorMsg(response));
  }

  return response.data.tag_sets;
}

export async function createTagSet(tags, antiTags) {
  let response = await apiCall("/api/v1/createTagSet", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      tags: tags,
      anti_tags: antiTags,
    }),
  });

  if (response.error_code !== 0) {
    error("Failed to create tag set: " + responseToErrorMsg(response));
  }

  return response.data.tag_set_id;
}

export async function changeTagSet(tagSetId, tags, antiTags) {
  let response = await apiCall("/api/v1/changeTagSet", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      tag_set_id: tagSetId,
      tags: tags,
      anti_tags: antiTags,
    }),
  });

  if (response.error_code !== 0) {
    error("Failed to change tag set: " + responseToErrorMsg(response));
  }
}

export async function deleteTagSet(tagSetId) {
  let response = await apiCall("/api/v1/deleteTagSet", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ tag_set_id: tagSetId }),
  });

  if (response.error_code !== 0) {
    error("Failed to delete tag set: " + responseToErrorMsg(response));
  }
}

export async function search(query, tags, antiTags, pageSize, pageNumber) {
  let response = await apiCall("/api/v1/search", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      query: query,
      tags: tags,
      anti_tags: antiTags,
      page_size: pageSize,
      page_number: pageNumber,
    }),
  });

  if (response.error_code !== 0) {
    error("Failed to search: " + responseToErrorMsg(response));
  }

  return response.data.results;
}

export async function getPage(pageId) {
  if (config.safe_mode) {
    return makeSafeModeImage(480, 668, pageId);
  }

  const response = await fetch(config.api_url + "/api/v1/page", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ page_id: pageId }),
  });

  if (response.headers.get("Content-Type").includes("application/json")) {
    let responseJson = await response.json();

    // If the endpoint returned json, this should never be 0, but just in case
    if (responseJson.error_code !== 0) {
      error("Failed to get page: " + responseToErrorMsg(responseJson));
    }
    throw "Unreachable";
  }

  const blob = await response.blob();
  const imgURL = URL.createObjectURL(blob);
  return imgURL;
}

export async function needsLogin() {
  const response = await apiCall("/api/v1/needsLogin", { method: "POST" });
  if (response.error_code !== 0) {
    error(
      "Failed to check if login is needed: " + responseToErrorMsg(response),
    );
  }

  return response.data.needs_login;
}

export async function getUsername() {
  const response = await apiCall("/api/v1/getUsername", { method: "POST" });
  if (response.error_code !== 0) {
    error("Failed to get username: " + responseToErrorMsg(response));
  }

  return response.data.username;
}

export async function getDoujin(doujinId) {
  const response = await apiCall("/api/v1/doujin", {
    method: "POST",
    body: JSON.stringify({ doujin_id: doujinId }),
  });
  if (response.error_code !== 0) {
    error("Failed to get doujin: " + responseToErrorMsg(response));
  }

  return response.data.doujin;
}
