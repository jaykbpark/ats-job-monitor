import { searchCompanies } from "./catalog.js";
import { parseAtsIdentifier } from "./ats.js";

const [command, ...rest] = process.argv.slice(2);

switch (command) {
  case "detect":
    runDetect(rest);
    break;
  case "companies":
    runCompanies(rest);
    break;
  default:
    printUsage(1);
}

function runDetect(args) {
  const input = args.join(" ").trim();

  if (!input) {
    printUsage(1);
  }

  const match = parseAtsIdentifier(input);

  if (!match) {
    console.error("Could not match a supported ATS URL.");
    process.exit(1);
  }

  console.log(JSON.stringify(match, null, 2));
}

function runCompanies(args) {
  const query = args.join(" ");
  console.log(JSON.stringify(searchCompanies(query), null, 2));
}

function printUsage(exitCode) {
  console.error("Usage:");
  console.error("  npm run detect -- <ats-url>");
  console.error("  npm run companies -- [query]");
  process.exit(exitCode);
}
