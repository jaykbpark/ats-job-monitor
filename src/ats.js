const PROVIDER_IDENTIFIER_KIND = {
  lever: "site",
  greenhouse: "board_token",
  ashby: "job_board_name"
};

export function getIdentifierKind(provider) {
  return PROVIDER_IDENTIFIER_KIND[provider] ?? "board_key";
}

export function buildBoardUrl(provider, boardKey) {
  switch (provider) {
    case "lever":
      return `https://jobs.lever.co/${boardKey}`;
    case "greenhouse":
      return `https://job-boards.greenhouse.io/${boardKey}`;
    case "ashby":
      return `https://jobs.ashbyhq.com/${boardKey}`;
    default:
      return null;
  }
}

export function buildApiUrl(provider, boardKey) {
  switch (provider) {
    case "lever":
      return `https://api.lever.co/v0/postings/${boardKey}?mode=json`;
    case "greenhouse":
      return `https://boards-api.greenhouse.io/v1/boards/${boardKey}/jobs`;
    case "ashby":
      return `https://api.ashbyhq.com/posting-api/job-board/${boardKey}`;
    default:
      return null;
  }
}

export function parseAtsIdentifier(input) {
  const url = coerceUrl(input);

  if (!url) {
    return null;
  }

  return parseLever(url) ?? parseGreenhouse(url) ?? parseAshby(url);
}

function coerceUrl(input) {
  if (typeof input !== "string") {
    return null;
  }

  const trimmed = input.trim();

  if (!trimmed) {
    return null;
  }

  try {
    return new URL(trimmed);
  } catch {
    try {
      return new URL(`https://${trimmed}`);
    } catch {
      return null;
    }
  }
}

function parseLever(url) {
  const host = url.hostname.toLowerCase();
  const pathParts = trimPath(url.pathname);

  if (host === "api.lever.co" && pathParts[0] === "v0" && pathParts[1] === "postings" && pathParts[2]) {
    return createMatch("lever", pathParts[2], url, "api-url");
  }

  if (host === "jobs.lever.co" || host === "jobs.eu.lever.co") {
    if (pathParts[0]) {
      return createMatch("lever", pathParts[0], url, "board-url");
    }
  }

  return null;
}

function parseGreenhouse(url) {
  const host = url.hostname.toLowerCase();
  const pathParts = trimPath(url.pathname);

  if (host === "boards-api.greenhouse.io" && pathParts[0] === "v1" && pathParts[1] === "boards" && pathParts[2]) {
    return createMatch("greenhouse", pathParts[2], url, "api-url");
  }

  if (host === "boards.greenhouse.io" && pathParts[0] === "embed" && pathParts[1] === "job_board") {
    const boardToken = url.searchParams.get("for");

    if (boardToken) {
      return createMatch("greenhouse", boardToken, url, "embed-url");
    }
  }

  if (host === "boards.greenhouse.io" || host === "job-boards.greenhouse.io") {
    if (pathParts[0]) {
      return createMatch("greenhouse", pathParts[0], url, "board-url");
    }
  }

  return null;
}

function parseAshby(url) {
  const host = url.hostname.toLowerCase();
  const pathParts = trimPath(url.pathname);

  if (host === "api.ashbyhq.com" && pathParts[0] === "posting-api" && pathParts[1] === "job-board" && pathParts[2]) {
    return createMatch("ashby", pathParts[2], url, "api-url");
  }

  if (host === "jobs.ashbyhq.com" && pathParts[0]) {
    return createMatch("ashby", pathParts[0], url, "board-url");
  }

  return null;
}

function createMatch(provider, boardKey, url, sourceType) {
  const normalizedKey = decodeURIComponent(boardKey).trim();

  return {
    provider,
    identifierKind: getIdentifierKind(provider),
    boardKey: normalizedKey,
    sourceType,
    normalizedUrl: url.toString(),
    boardUrl: buildBoardUrl(provider, normalizedKey),
    apiUrl: buildApiUrl(provider, normalizedKey)
  };
}

function trimPath(pathname) {
  return pathname.split("/").filter(Boolean);
}
