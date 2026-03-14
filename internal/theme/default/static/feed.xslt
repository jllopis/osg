<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet version="1.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:atom="http://www.w3.org/2005/Atom"
  exclude-result-prefixes="atom">
<xsl:output method="html" encoding="utf-8" indent="yes" />

<xsl:template match="/">
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>
    <xsl:choose>
      <xsl:when test="/rss/channel/title"><xsl:value-of select="/rss/channel/title" /> — RSS Feed</xsl:when>
      <xsl:when test="/atom:feed/atom:title"><xsl:value-of select="/atom:feed/atom:title" /> — Atom Feed</xsl:when>
      <xsl:otherwise>Web Feed</xsl:otherwise>
    </xsl:choose>
  </title>
  <style>
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:"Inter",system-ui,-apple-system,sans-serif;line-height:1.6;
      background:#2e3440;color:#d8dee9;max-width:48rem;margin:0 auto;padding:2rem 1rem}
    a{color:#88c0d0;text-decoration:none}
    a:hover{text-decoration:underline;color:#8fbcbb}
    .feed-header{border-bottom:1px solid #4c566a;padding-bottom:1.5rem;margin-bottom:2rem}
    .feed-header h1{font-size:1.5rem;color:#eceff4;margin-bottom:.5rem}
    .feed-header p{color:#81a1c1;font-size:.9rem}
    .feed-header .badge{display:inline-block;background:#5e81ac;color:#eceff4;
      font-size:.75rem;padding:.15rem .5rem;border-radius:.25rem;margin-right:.5rem;
      vertical-align:middle;font-weight:600;text-transform:uppercase}
    .feed-meta{color:#81a1c1;font-size:.85rem;margin-top:.5rem}
    .feed-meta code{background:#3b4252;padding:.1rem .35rem;border-radius:.2rem;
      font-size:.8rem;font-family:monospace;color:#a3be8c;word-break:break-all}
    .entry{border-bottom:1px solid #3b4252;padding:1.25rem 0}
    .entry:last-child{border-bottom:none}
    .entry h2{font-size:1.1rem;color:#eceff4;margin-bottom:.25rem}
    .entry .date{color:#81a1c1;font-size:.8rem}
    .entry .summary{color:#d8dee9;margin-top:.5rem;font-size:.9rem}
    @media(prefers-color-scheme:light){
      body{background:#eceff4;color:#2e3440}
      .feed-header{border-color:#d8dee9}
      .feed-header h1{color:#2e3440}
      .feed-header p,.feed-meta,.entry .date{color:#4c566a}
      .feed-meta code{background:#d8dee9;color:#2e3440}
      .entry{border-color:#d8dee9}
      .entry h2{color:#2e3440}
      .entry .summary{color:#3b4252}
      a{color:#5e81ac}a:hover{color:#81a1c1}
    }
  </style>
</head>
<body>
  <div class="feed-header">
    <h1>
      <xsl:choose>
        <xsl:when test="/rss"><span class="badge">RSS</span></xsl:when>
        <xsl:when test="/atom:feed"><span class="badge">Atom</span></xsl:when>
      </xsl:choose>
      <xsl:choose>
        <xsl:when test="/rss/channel/title"><xsl:value-of select="/rss/channel/title" /></xsl:when>
        <xsl:when test="/atom:feed/atom:title"><xsl:value-of select="/atom:feed/atom:title" /></xsl:when>
      </xsl:choose>
    </h1>
    <p>This is a web feed. Subscribe by copying the URL into your feed reader.</p>
    <xsl:if test="/rss/channel/description">
      <p><xsl:value-of select="/rss/channel/description" /></p>
    </xsl:if>
    <xsl:if test="/atom:feed/atom:subtitle">
      <p><xsl:value-of select="/atom:feed/atom:subtitle" /></p>
    </xsl:if>
  </div>

  <!-- RSS items -->
  <xsl:for-each select="/rss/channel/item">
    <div class="entry">
      <h2><a href="{link}"><xsl:value-of select="title" /></a></h2>
      <div class="date"><xsl:value-of select="pubDate" /></div>
      <xsl:if test="description">
        <div class="summary"><xsl:value-of select="description" /></div>
      </xsl:if>
    </div>
  </xsl:for-each>

  <!-- Atom entries -->
  <xsl:for-each select="/atom:feed/atom:entry">
    <div class="entry">
      <h2><a href="{atom:link/@href}"><xsl:value-of select="atom:title" /></a></h2>
      <div class="date"><xsl:value-of select="atom:updated" /></div>
      <xsl:if test="atom:summary">
        <div class="summary"><xsl:value-of select="atom:summary" /></div>
      </xsl:if>
    </div>
  </xsl:for-each>

</body>
</html>
</xsl:template>
</xsl:stylesheet>
