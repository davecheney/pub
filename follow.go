package main

type FollowCmd struct {
	Object string `help:"object to follow" required:"true"`
	Actor  string `help:"actor to follow with" required:"true"`
}

func (f *FollowCmd) Run(ctx *Context) error {
	// db, err := gorm.Open(ctx.Dialector, &ctx.Config)
	// if err != nil {
	// 	return err
	// }
	return nil

	// var account models.Account
	// if err := db.Joins("Actor", &models.Actor{URI: f.Actor}).Take(&account).Error; err != nil {
	// 	return err
	// }

	// return activitypub.Follow(context.Background(), &account, &models.Actor{URI: f.Object})
}
