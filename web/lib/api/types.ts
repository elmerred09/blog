import type { components } from "./schema";

type Schemas = components["schemas"];

export type Tag = Schemas["Tag"];
export type PostListItem = Schemas["PostListItem"];
export type Post = Schemas["Post"];
export type Page = Schemas["Page"];
export type SitemapEntry = Schemas["SitemapEntry"];
export type SitemapPage = Schemas["SitemapPage"];
export type SnippetFragment = Schemas["SnippetFragment"];
export type SearchResult = Schemas["SearchResult"];
export type SearchResults = Schemas["SearchResults"];
export type ErrorCode = Schemas["Error"]["error"]["code"];
export type Problem = NonNullable<
  Schemas["Error"]["error"]["problems"]
>[number];

export type AdminPage = Schemas["AdminPage"];
export type AdminPostListItem = Schemas["AdminPostListItem"];
export type AdminPost = Schemas["AdminPost"];
export type AdminPostInput = Schemas["AdminPostInput"];
export type AdminPostUpdate = Schemas["AdminPostUpdate"];
export type Upload = Schemas["Upload"];
export type UploadInput = Schemas["UploadInput"];
