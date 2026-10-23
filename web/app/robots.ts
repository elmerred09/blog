import type { MetadataRoute } from "next";

import { buildRobots } from "@/lib/robots";
import { indexingEnabled, siteUrl } from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  return buildRobots(indexingEnabled(), siteUrl());
}
