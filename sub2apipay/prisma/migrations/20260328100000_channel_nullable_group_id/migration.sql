-- AlterTable: make channel group_id nullable
ALTER TABLE "channels" ALTER COLUMN "group_id" DROP NOT NULL;
