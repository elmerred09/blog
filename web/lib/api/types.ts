import type { components } from "./schema";

type Schemas = components["schemas"];

export type Tag = Schemas["Tag"];
export type PostListItem = Schemas["PostListItem"];
export type Post = Schemas["Post"];
export type Page = Schemas["Page"];
export type SnippetFragment = Schemas["SnippetFragment"];
export type SearchResult = Schemas["SearchResult"];
export type SearchResults = Schemas["SearchResults"];
export type ErrorCode = Schemas["Error"]["error"]["code"];
