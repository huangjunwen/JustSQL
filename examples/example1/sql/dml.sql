-- $func:UsersByIds
-- $arg:userIds type:[]int
-- $env hasInBinding:true
SELECT * FROM user WHERE id IN (/*$bind:userIds*/1/**/);

-- $func:QueryBlogById return:one
-- $arg:blogId type:int
SELECT b.*, u.nick
FROM blog b, user u
WHERE b.user_id=u.id AND b.id=/*$bind:blogId*/1/**/;

-- $func:QueryBlog return:many
-- $arg:userNick type:string
-- $arg:title type:string
SELECT *
FROM blog b JOIN user u ON (b.user_id=u.id)
WHERE 1
    /*$${{ if ne .userNick "" }}*/AND u.nick=/*$bind:userNick*/"jayven"/**/ /*$${{ end }}*/
    /*$${{ if ne .title "" }}*/AND b.title=/*$bind:title*/"How to use JustSQL?"/**/ /*$${{ end }}*/;

